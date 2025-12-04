# Home Tidal Flood Warning

My home located quite near to the sea, and mine's placed a little bit below road level,
as it was built more than 30 years ago. At monsoon season like this, if we had heavy rain
and the tide level is high, we are quite prone to get flash-flooded. This application is built to
alert our household about the possibility / risk of flooding.

## Based On

This project is built on top of [weather-alert](https://github.com/shadowbane/weather-alert), a Go REST API for fetching and serving BMKG weather alerts.
It extends the base module using Go struct embedding to add tidal flood warning capabilities while reusing the weather alert functionality.

## Data Source & Attribution

Weather alert data is sourced from **BMKG (Badan Meteorologi, Klimatologi, dan Geofisika)** - Indonesia's Meteorological, Climatological, and Geophysical Agency.

Data follows the CAP (Common Alerting Protocol) format as provided by BMKG:
- **BMKG CAP Data Repository**: https://github.com/infoBMKG/data-cap

### Disclaimer

This is an unofficial personal project and is not affiliated with or endorsed by BMKG. All weather alert data remains the property of BMKG. Please refer to BMKG's official channels for authoritative weather information.

## License

This project is for personal/educational use. Weather data is provided by BMKG under their terms of use.
