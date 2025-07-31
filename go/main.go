package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"context"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Inmueble struct {
	Title       string
	Price       string
	Location    string
	Address     string
	DistanceM   int
	Details     string
	Link        string
	Descripcion string
}

type GeoLocation struct {
	Lat float64
	Lng float64
}

func exportToGoogleSheets(spreadsheetID string, sheetName string, inmuebles []Inmueble, credentialsPath string) error {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return fmt.Errorf("no se pudo crear el cliente de Sheets: %v", err)
	}

	// Encabezados
	var data [][]interface{}
	data = append(data, []interface{}{"Titulo", "Precio", "Dirección", "Detalles", "Link", "Descripción"})

	for _, i := range inmuebles {
		data = append(data, []interface{}{
			i.Details, i.Price, i.Address, i.Location, i.Link, i.Descripcion,
		})
	}

	writeRange := fmt.Sprintf("%s!A1", sheetName)
	valueRange := &sheets.ValueRange{
		Range:  writeRange,
		Values: data,
	}

	_, err = srv.Spreadsheets.Values.Update(spreadsheetID, writeRange, valueRange).
		ValueInputOption("RAW").
		Do()
	if err != nil {
		return fmt.Errorf("no se pudo escribir en la hoja: %v", err)
	}

	return nil
}
func getCoordinates(address, apiKey string) (*GeoLocation, error) {
	baseURL := "https://maps.googleapis.com/maps/api/geocode/json"
	params := url.Values{}
	params.Set("address", address)
	params.Set("key", apiKey)

	resp, err := http.Get(fmt.Sprintf("%s?%s", baseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Results []struct {
			Geometry struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	if result.Status != "OK" || len(result.Results) == 0 {
		return nil, fmt.Errorf("geocoding failed: %s", result.Status)
	}

	loc := result.Results[0].Geometry.Location
	return &GeoLocation{Lat: loc.Lat, Lng: loc.Lng}, nil
}

func distanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1 = lat1 * math.Pi / 180
	lat2 = lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func fetchInmuebles(url string, ch chan<- Inmueble, wg *sync.WaitGroup, baseLat, baseLng float64, radius float64) {
	defer wg.Done()
	resp, err := http.Get(url)
	if err != nil {
		log.Println("❌ Error en", url, err)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Println("❌ Error al parsear HTML:", err)
		return	
	}

	elements := doc.Find("div.listingCard")
	count := elements.Length()
	fmt.Printf("🔍 Procesando página %s... encontró %d inmuebles\n", url, count)
	if count == 0 {
		fmt.Printf("⚠️ documento %s", doc)
	}

	elements.Each(func(i int, s *goquery.Selection) {
		title := s.Find("span.lc-title").Text()
		location := s.Find("strong.lc-location").Text()
		link, _ := s.Find("a.lc-data").Attr("href")
		price := s.Find("div.lc-price").Text()
		details := strings.TrimSpace(s.Find("div.lc-typologyTag").Text())

		fullLink := "https://www.fincaraiz.com.co" + link
		lat, lng, desc, addr := fetchDetalle(fullLink)
		// fmt.Printf("🔗 Detalle: %s %f %f\n", title, lat, lng)
		
		if lat == 0 || lng == 0 {
			return
		}
		dist := distanceMeters(baseLat, baseLng, lat, lng)
		// fmt.Printf("🔍 Inmueble: %s, Distancia: %.2f m\n", title, dist)
		if dist > radius {
			return
		}

		ch <- Inmueble{
			Title:       strings.TrimSpace(title),
			Price:       strings.TrimSpace(price),
			Location:    strings.TrimSpace(location),
			Address:     addr,
			DistanceM:   int(dist),
			Details:     strings.TrimSpace(details),
			Link:        fullLink,
			Descripcion: desc,
		}
	})
}

func fetchDetalle(url string) (float64, float64, string, string) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, "", ""
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, 0, "", ""
	}

	desc := strings.TrimSpace(doc.Find("div.property-description span").Text())
	var lat, lng float64
	var addr string

	doc.Find("script[type='application/ld+json']").EachWithBreak(func(i int, s *goquery.Selection) bool {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(s.Text()), &data); err != nil {
			return true 
		}

		// for key, value := range data {
		// 	fmt.Printf("  %s: %v\n", key, value)
		// }

		var geo map[string]interface{}

		if g, ok := data["geo"].(map[string]interface{}); ok {
			geo = g
		} else if obj, ok := data["object"].(map[string]interface{}); ok {
			if g, ok := obj["geo"].(map[string]interface{}); ok {
				geo = g
			}
			if a, ok := obj["address"].(string); ok {
				addr = a
			}
		}

		if geo != nil {
			if latVal, ok := geo["latitude"].(float64); ok {
				lat = latVal
			}
			if lngVal, ok := geo["longitude"].(float64); ok {
				lng = lngVal
			}
		}

		if addr == "" {
			if a, ok := data["address"].(string); ok {
				addr = a
			}
		}

		if lat != 0 && lng != 0 {
			return false
		}
		return true
	})

	return lat, lng, desc, addr
}

func saveCSV(filename string, inmuebles []Inmueble) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"title", "price", "location", "address", "distancia_m", "details", "link", "descripcion"})
	for _, i := range inmuebles {
		writer.Write([]string{
			i.Title, i.Price, i.Location, i.Address,
			fmt.Sprintf("%d", i.DistanceM), i.Details, i.Link, i.Descripcion,
		})
	}
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No se encontró el archivo .env, cargando variables de entorno por defecto")
	}
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")


	var direccionBase, tipo, ciudad, localidad, modalidad string
	var radio float64 = 1000

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("📍 Dirección base: ")
	// direccionBase = "calle 107b #78c-45, Bogotá, Colombia" // Default address for testing
	direccionBase, _ = reader.ReadString('\n')
	direccionBase = strings.TrimSpace(direccionBase)

	fmt.Print("🏠 Tipo (apartamentos/casas): ")
	tipo, _ = reader.ReadString('\n')
	tipo = strings.TrimSpace(tipo)
	// tipo = "apartamentos" // Default type for testing

	fmt.Print("🌆 Ciudad: ")
	ciudad, _ = reader.ReadString('\n')
	ciudad = strings.TrimSpace(ciudad)
	// ciudad = "bogota" // Default city for testing

	
	fmt.Print("📍 Localidad: ")
	localidad, _ = reader.ReadString('\n')
	localidad = strings.TrimSpace(localidad)
	// localidad = "engativa" // Default locality for testing

	fmt.Print("📏 Radio en metros: ")
	fmt.Fscanln(reader, &radio)

	fmt.Print("📈 Modalidad (arriendo/venta): ")
	modalidad, _ = reader.ReadString('\n')
	modalidad = strings.TrimSpace(modalidad)
	// modalidad = "venta"

	geo, err := getCoordinates(direccionBase, apiKey)
	if err != nil {
		log.Fatal("❌ Error obteniendo coordenadas:", err)
	}

	fmt.Printf("✅ Coordenadas: %.6f, %.6f\n", geo.Lat, geo.Lng)

	baseURL := fmt.Sprintf("https://www.fincaraiz.com.co/%s/%s/%s/%s", modalidad, tipo, localidad, ciudad)
	fmt.Printf("🔗 URL base: %s\n", baseURL)
	ch := make(chan Inmueble, 100)
	var wg sync.WaitGroup
	inmuebles := []Inmueble{}

	for i := 1; i <= 20; i++ {
		wg.Add(1)
		url := fmt.Sprintf("%s/pagina%d", baseURL, i)
		go fetchInmuebles(url, ch, &wg, geo.Lat, geo.Lng, radio)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()
	
	for i := range ch {
		inmuebles = append(inmuebles, i)
	}

	fmt.Printf("✅ Se encontraron %d inmuebles\n", len(inmuebles))

	if err := saveCSV("resultados_fincaraiz.csv", inmuebles); err != nil {
		log.Fatal("❌ Error guardando CSV:", err)
	}
	fmt.Println("📁 Resultados guardados en resultados_fincaraiz.csv")

		// 👇 Exportar a Google Sheets
	credentialsPath := "credenciales.json"
	spreadsheetID := "14b_yUClaYbMIipp7axnigDM1YBeEci25QI4yJ1wXYqA"
	sheetName := "Hoja Go"

	if err := exportToGoogleSheets(spreadsheetID, sheetName, inmuebles, credentialsPath); err != nil {
		log.Fatal("❌ Error exportando a Google Sheets:", err)
	}
	fmt.Println("✅ Datos exportados a Google Sheets")
}