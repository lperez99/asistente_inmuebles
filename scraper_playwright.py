import asyncio
from playwright.async_api import async_playwright
import json

async def main():
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True) 
        context = await browser.new_context()
        page = await context.new_page()

        # Interceptar respuesta a la API de búsqueda
        async def handle_response(response):
            if "properties/search" in response.url and response.request.method == "POST":
                try:
                    json_data = await response.json()
                    hits = json_data["data"]["search"]["hits"]
                    print("✅ Resultados encontrados:")
                    for hit in hits:
                        info = hit["_source"]["Listing"]
                        print(f"Título: {info['title']}")
                        print(f"Dirección: {info['address']}")
                        print(f"Precio: {info['price'].get('sale') or info['price'].get('rent')}")
                        print(f"Link: https://www.fincaraiz.com.co/inmueble/{info['slug']}")
                        print("------")
                except:
                    print("No se pudo parsear la respuesta.")

        page.on("response", handle_response)

        # Abre directamente la búsqueda que devuelve inmuebles
        await page.goto("https://www.fincaraiz.com.co/arriendo/apartamentos/bogota-dc")
        
        # Espera suficiente tiempo para que cargue el request
        await page.wait_for_timeout(10000)

        await browser.close()

asyncio.run(main())
