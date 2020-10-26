# covid-19-api

## Routes

Base URL : <b>covid19trackingapi.com</b>

Paths : <br><br>
<b>/list/countries</b> : Returns a list of countries with their name and slug. Please use the country slug when requesting for data for a specific country.<br><br>
<b>/global</b> : Returns the number of confirmed cases, recoveries, and deaths globally.<br><br>
<b>/summary</b> : Returns the number of confirmed cases, recoveries, and deaths both globally and per country.<br><br>
<b>/timeseries/{countryslug}/{status}</b> : Returns the history of either confirmed cases, recoveries, and deaths of the 
specified country and each of its provinces starting from Jan. 22, 2020. {countryslug} <b>must</b> be a valid country slug from '/list/countries'. 
{status} <b>must</b> be one of the following: [confirmed, recoveries, deaths].<br><br>
<b>/timeseries/total/{countryslug}/{status}</b> : Returns the history of either confirmed cases, recoveries, and deaths 
of the specified country starting from Jan. 22, 2020. Unlike '/timeseries/{countryslug}/{status}', this route does not return a country's provinces. 
Instead, the data is all summed up. {countryslug} <b>must</b> be a valid country slug from '/list/countries'. {status} <b>must</b> be one of the following: [confirmed, recoveries, deaths].
## Run locally

Note: Make sure [Docker](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/install/) are installed.

1. Create a new environment file named `.env` using your preferred text editor  (use .env.example as an example).

2. Run with docker compose. <br>
`docker-compose up`

3. Go to `localhost:{port}/summary` to verify that the service is up. It may take some seconds to seed the database.
