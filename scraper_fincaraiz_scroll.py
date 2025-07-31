import asyncio
import json
import re
from playwright.async_api import async_playwright

URL = "https://www.fincaraiz.com.co/arriendo/apartamentos/bogota/bogota-dc"

async def run():
    async with async_playwright() as p:
        #Abrir el navegador 
        browser = await p.chromium.launch(headless=True)
        page = await browser.new_page()
        await page.goto(URL)

        await page.wait_for_selector("div.listingCard.highlight", timeout=15000)
        cards = await page.query_selector_all("div.listingCard.highlight")

        for card in cards:
            # Título
            title_span = await card.query_selector("span.lc-title")
            title = await title_span.inner_text() if title_span else "Sin título"

            # Ubicación
            location_tag = await card.query_selector("strong.lc-location")
            location = await location_tag.inner_text() if location_tag else "Ubicación no encontrada"

            # Enlace
            link_tag = await card.query_selector("a.lc-data")
            link = await link_tag.get_attribute("href") if link_tag else ""

            # Precio
            price_tag = await card.query_selector("div.lc-price")
            price = await price_tag.inner_text() if price_tag else "Precio no encontrado"

            # Detalles
            details_tag = await card.query_selector("div.lc-typologyTag")
            details_text = await details_tag.inner_text() if details_tag else "Detalles no encontrados"
            details_text = re.sub(r"\s+", " ", details_text).strip()
            details_text = re.sub(r"\b(Baños|Baño)(\d)", r"\1 \2", details_text)
            details_text = re.sub(r"(\d)(m²)", r"\1 \2", details_text)

            # Ir a la página del inmueble para obtener coordenadas y dirección
            lat, lon, address = None, None, None
            if link:
                detail_page = await browser.new_page()
                await detail_page.goto(f"https://www.fincaraiz.com.co{link}")

                try:
                    await detail_page.wait_for_selector("body", timeout=10000)
                    scripts = await detail_page.query_selector_all('script[type="application/ld+json"]')

                    for index in range(len(scripts)):
                        script = scripts[index]
                        content = await script.inner_text()

                        try:
                            json_data = json.loads(content)

                            # Algunos JSONs pueden ser listas
                            if isinstance(json_data, list):
                                items = json_data
                            else:
                                items = [json_data]

                            for item in items:
                                geo_data = item.get("object")
                                if geo_data:
                                    address = geo_data.get("address")
                                    lat_long_data = geo_data.get("geo")

                                    lat = lat_long_data.get("latitude") if lat_long_data else None
                                    lon = lat_long_data.get("longitude") if lat_long_data else None

                        except json.JSONDecodeError as e:
                            print(f"❌ Error parseando JSON del script #{index+1}:", e)

                except Exception as e:
                    print("⚠️ Error cargando detalle general:", e)


                await detail_page.close()

            # Mostrar información
            print("🏠 Inmueble")
            print(f"Título: {title}")
            print(f"Ubicación: {location}")
            print(f"Precio: {price}")
            print(f"Detalles: {details_text}")
            print(f"Dirección completa: {address}")
            print(f"Latitud: {lat}, Longitud: {lon}")
            print(f"Link: https://www.fincaraiz.com.co{link}")
            print("––––––––––––––––––––––––")

        await browser.close()

asyncio.run(run())
