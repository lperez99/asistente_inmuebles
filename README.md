# 🏘️ Asistente Inmuebles – Web Scraper + Google Sheets + Geolocalización 📍

Este proyecto es un **asistente virtual para encontrar inmuebles** en arriendo o venta 🏠 en Colombia, de forma automática, rápida y personalizada. Está diseñado para **automatizar la búsqueda**, aplicar filtros por distancia y guardar los resultados en una hoja de cálculo. Ideal para quienes buscan vivienda sin perder tiempo buscando en muchas páginas manualmente.

---

## 🚀 ¿Qué hace este proyecto?

1. 🔍 **Scrapea propiedades** desde el portal web [Finca Raíz](https://fincaraiz.com.co) 
2. 🧠 **Filtra resultados** según tus criterios (ubicación, tipo, precio, etc).
3. 🗺️ **Calcula la distancia** desde cada propiedad a un punto de interés (por ejemplo, tu oficina).
4. 📊 **Guarda los resultados** automáticamente en una hoja de cálculo de Google Sheets.
5. 🤖 Puede ser usado dentro de un flujo de automatización como [n8n](https://n8n.io).

---

## 🛠️ Tecnologías utilizadas

- 🐍 Python
- 🌐 Playwright para scraping
- 📍 Google Maps API (distancia geográfica)
- 📄 Google Sheets API
- 📦 Concurrente y optimizado con `asyncio` o `Go` (en versiones avanzadas)

---

## 📁 Estructura del proyecto

```
asistente_inmuebles/
├── scraper_playwright.py        # Scraper principal usando Playwright
├── scraper_fincaraiz_scroll_geo.py  # Scraper para Finca Raíz con scroll y geolocalización
├── sheets.py                    # Módulo para enviar datos a Google Sheets
├── geolocalizador.py           # Obtiene lat/lon y distancia
├── data/                       # Carpeta donde se guardan resultados temporales
└── README.md                   # Este archivo 😉
```

---

## ⚙️ Cómo usarlo

1. Clona este repositorio:

```bash
git clone https://github.com/tu-usuario/asistente_inmuebles.git
cd asistente_inmuebles
```

2. Crea tu archivo `.env` con las siguientes variables:

```
GOOGLE_API_KEY=...
GOOGLE_SHEET_ID=...
SHEET_NAME=...
```

3. Instala las dependencias:

```bash
pip install -r requirements.txt
```

4. Ejecuta el scraper que necesites:

```bash
python scraper_playwright.py
```

---

## 🧠 Personaliza tu búsqueda

Puedes modificar tus filtros directamente en el código o pasar parámetros según tus necesidades:

- Precio mínimo / máximo
- Tipo de propiedad
- Ciudad o barrio
- Radio en kilómetros desde un punto específico

---

## ✨ Funcionalidades futuras

- Notificaciones por email o Telegram cuando se detecten nuevas propiedades 📬
- Interfaz web para manejar filtros sin tocar código 🖥️
- Versión 100% en Go para mejorar rendimiento ⚡
- Agente conversacional con IA 🤖

---

## 🙋‍♀️ Autora

Proyecto desarrollado por **Laura Pérez**, como asistente personal y ejercicio técnico para automatización y scraping de datos inmobiliarios 🧩.

---

## 💌 Licencia

MIT License. ¡Úsalo, modifícalo y comparte si te ayuda!
