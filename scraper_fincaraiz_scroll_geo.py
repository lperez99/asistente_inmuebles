import asyncio
import json
import re
from playwright.async_api import async_playwright
import os
from dotenv import load_dotenv
import requests
from math import radians, sin, cos, sqrt, atan2
import gspread 
from oauth2client.service_account import ServiceAccountCredentials

load_dotenv()

TIPO_INMUEBLE = os.getenv("TIPO_INMUEBLE", "apartamentos")
direccion_base = input("📍 Dirección base para la búsqueda (ej: Calle 167 56–25, Bogotá): ").strip()
property_type = input("🔍 Tipo de propiedad: (apartamentos/casas)").lower()
city = input("🔍 Ciudad: ").lower()
localidad = input("🔍 Localidad: ").lower()
radio_m = float(input("🔍 Radio de búsqueda en metros (ej: 1000): ").strip())
modalidad = input("🔍 Modalidad de búsqueda (arriendo/venta): ").strip().lower()
URL = f"https://www.fincaraiz.com.co/{modalidad}/{property_type}/{localidad}/{city}"
API_KEY = os.getenv("GOOGLE_MAPS_API_KEY")

def get_coordinates(address, api_key):
    url = "https://maps.googleapis.com/maps/api/geocode/json"
    params = {"address": address, "key": api_key}
    try:
        resp = requests.get(url, params=params, timeout=10)
        resp.raise_for_status()
        data = resp.json()
        if data["status"] == "OK":
            loc = data["results"][0]["geometry"]["location"]
            return loc["lat"], loc["lng"]
        print("⚠️ Google no devolvió OK:", data.get("error_message", data["status"]))
    except requests.exceptions.RequestException as e:
        print("❌ Error de red:", e)
    return None

lat_base, lng_base = get_coordinates(direccion_base, API_KEY)
if not lat_base or not lng_base:
    raise SystemExit("No se pudo obtener coordenadas de la dirección base.")
print(f"📍 Coordenadas de la dirección base: {lat_base}, {lng_base}")

def distance_m(lat1, lon1, lat2, lon2):
    R = 6371000  # Radio en metros
    dlat = radians(lat2 - lat1)
    dlon = radians(lon2 - lon1)
    a = sin(dlat/2)**2 + cos(radians(lat1)) * cos(radians(lat2)) * sin(dlon/2)**2
    c = 2 * atan2(sqrt(a), sqrt(1 - a))
    return R * c

async def run():
    results = []
    async with async_playwright() as p: 
        browser = await p.chromium.launch(headless=True)
        page = await browser.new_page()

        for page_num in range(1, 20):  # Máximo 50 páginas
            paginated_url = f"{URL}/pagina{page_num}"
            print(f"\n📄 Página {page_num}: {paginated_url}")
            try:
                await page.goto(paginated_url, wait_until="domcontentloaded", timeout=60000)
            except Exception as e:
                print(f"❌ Error al cargar la página {page_num}: {e}")
                continue


            try:
                await page.wait_for_selector("div.listingCard.highlight", timeout=10000)
            except:
                print("⚠️ No se encontraron tarjetas en esta página.")
                break

            cards = await page.query_selector_all("div.listingCard.highlight")
            if not cards:
                print("✅ Fin del scraping, no hay más resultados.")
                break

            print(f"➡️ {len(cards)} inmuebles encontrados en esta página")

            for card in cards:
                title = await (await card.query_selector("span.lc-title")).inner_text()
                location = await (await card.query_selector("strong.lc-location")).inner_text()
                link = await (await card.query_selector("a.lc-data")).get_attribute("href")
                price = await (await card.query_selector("div.lc-price")).inner_text()
                details_tag = await card.query_selector("div.lc-typologyTag")
                details = await details_tag.inner_text() if details_tag else ""
                details = re.sub(r"\s+", " ", details).strip()

                lat = lon = address = descripcion = None
                url_detalle = f"https://www.fincaraiz.com.co{link}" if not link.startswith("http") else link
                detail_page = await browser.new_page()
                try:
                    await detail_page.goto(url_detalle, wait_until="domcontentloaded", timeout=60000)
                    desc_tag = await detail_page.query_selector("div.property-description span")
                    descripcion = await desc_tag.inner_text() if desc_tag else "Descripción no encontrada"
                    scripts = await detail_page.query_selector_all('script[type="application/ld+json"]')
                    for script in scripts:
                        try:
                            content = await script.inner_text()
                            data = json.loads(content)
                            items = data if isinstance(data, list) else [data]
                            for item in items:
                                geo = item.get("object", {}).get("geo") or item.get("geo")
                                if geo:
                                    lat = float(geo.get("latitude"))
                                    lon = float(geo.get("longitude"))
                                    address = item.get("object", {}).get("address") or item.get("address")
                                    break
                            if lat and lon:
                                break
                        except:
                            continue
                except Exception as e:
                    print("❌ Error cargando detalle:", e)
                await detail_page.close()

                if lat is None or lon is None:
                    continue

                distancia = distance_m(lat_base, lng_base, lat, lon)
                print(f"📐 Distancia: {distancia/1000:.2f} km ({distancia:.0f} m)")
                if distancia > radio_m:
                    continue

                print("🏢 Inmueble")
                print(f"🧾 Título: {title}")
                print(f"🤑 Precio: {price}")
                print(f"📍 Ubicación: {location}")
                print(f"📍 Dirección completa: {address}")
                print(f"📐 Distancia: {distancia/1000:.2f} km ({distancia:.0f} m)")
                print(f"🧱 Detalles: {details}")
                print(f"🔗 Link: {url_detalle}")
                print(f"📝 Descripción: {descripcion}")
                print("––––––––––––––––––––––––")
                

                

                # 👇 Dentro del for card in cards: después de calcular todo
                results.append([
                    title,
                    price,
                    location,
                    address,
                    int(distancia),
                    details,
                    url_detalle,
                    descripcion
                ])
                print("✅ Inmueble guardado correctamente.")


        await browser.close()
        
        # ✨ Subir a Google Sheets
    scope = ["https://spreadsheets.google.com/feeds", "https://www.googleapis.com/auth/drive"]
    creds = ServiceAccountCredentials.from_json_keyfile_name("credenciales.json", scope)
    client = gspread.authorize(creds)

    sheet = client.open("Resultados Finca Raíz")
    worksheet = sheet.sheet1
    # worksheet = sheet.get_worksheet(0)
    worksheet.update(
        "A1",
        [["title", "price", "location", "address", "distancia_m", "details", "link", "descripcion"]] + results
    )
    print("📤 Resultados guardados en Google Sheets correctamente.")


asyncio.run(run())
