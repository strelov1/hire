package location

// The dictionaries below are seeded from the high-frequency location strings
// observed in production ATS data. They are meant to grow by observation — add
// the names/cities that show up unresolved, not a full gazetteer up front.

// regionCountries groups ISO 3166-1 alpha-2 country codes under one canonical
// region code from enrich.RegionValues. Each country maps to exactly one region
// (the coarse facet a user filters on); countryToRegion is the inverted lookup.
// "eu" is used in the broad geographic sense of Europe (not only EU members).
var regionCountries = map[string][]string{
	"eu": {
		"de", "fr", "nl", "es", "se", "pl", "ie", "pt", "it", "be", "dk",
		"fi", "at", "cz", "ro", "gr", "hu", "bg", "hr", "sk", "si", "lt",
		"lv", "ee", "lu", "ch", "no", "ua", "is",
	},
	"uk":            {"gb"},
	"us":            {"us"},
	"north_america": {"ca"},
	"latam":         {"ar", "br", "mx", "cl", "co", "pe", "uy"},
	"apac":          {"sg", "jp", "au", "nz", "in", "hk", "tw", "kr", "cn", "my", "th", "ph", "vn", "id"},
	"mena":          {"ae", "sa", "il", "eg", "tr", "qa"},
	"africa":        {"za", "ng", "ke"},
	"ru":            {"ru"},
	// CIS / Central Asia — the RU-segment geography of the Telegram sources.
	// central_asia is the five republics; cis is the rest of the post-Soviet
	// space (Belarus, Moldova, the Caucasus). ru keeps its own area; ua stays eu.
	"central_asia": {"uz", "kz", "kg", "tj", "tm"},
	"cis":          {"by", "md", "am", "az", "ge"},
}

// countryToRegion is the inverted regionCountries: ISO code -> region code.
var countryToRegion = invertRegionCountries()

func invertRegionCountries() map[string]string {
	out := make(map[string]string)
	for region, codes := range regionCountries {
		for _, code := range codes {
			out[code] = region
		}
	}
	return out
}

// nameToCountry resolves lowercase country names, common ATS shorthands, and a
// few beacon cities to an ISO 3166-1 alpha-2 code. The region falls out of
// countryToRegion, so shorthands like "uk" yield both the country (gb) and its
// region (uk) without a separate entry.
var nameToCountry = map[string]string{
	"united states": "us", "united states of america": "us",
	"usa": "us", "us": "us", "u.s.": "us", "u.s.a.": "us",
	"united kingdom": "gb", "uk": "gb", "u.k.": "gb",
	"england": "gb", "britain": "gb", "great britain": "gb", "london": "gb",
	"germany": "de", "deutschland": "de", "berlin": "de", "munich": "de", "münchen": "de", "hamburg": "de",
	"france": "fr", "paris": "fr",
	"netherlands": "nl", "the netherlands": "nl", "amsterdam": "nl",
	"spain": "es", "madrid": "es", "barcelona": "es",
	"sweden": "se", "stockholm": "se",
	"poland": "pl", "warsaw": "pl",
	"ireland": "ie", "dublin": "ie",
	"portugal": "pt", "lisbon": "pt",
	"italy": "it", "milan": "it", "rome": "it",
	"belgium": "be", "brussels": "be",
	"denmark": "dk", "copenhagen": "dk",
	"finland": "fi", "helsinki": "fi",
	"austria": "at", "vienna": "at",
	"switzerland": "ch", "zurich": "ch",
	"norway": "no", "ukraine": "ua",
	"canada": "ca", "toronto": "ca", "vancouver": "ca", "montreal": "ca",
	"singapore": "sg",
	"australia": "au", "sydney": "au", "melbourne": "au",
	"new zealand": "nz",
	"japan":       "jp", "tokyo": "jp",
	"india": "in", "pune": "in", "bangalore": "in", "bengaluru": "in", "mumbai": "in", "hyderabad": "in",
	"argentina": "ar", "brazil": "br", "mexico": "mx",
	"israel": "il", "tel aviv": "il",
	"united arab emirates": "ae", "dubai": "ae",
	"south africa": "za",
	// RU / CIS / Central Asia. "georgia" is deliberately absent — it collides with
	// the US state; the country resolves via its capital "tbilisi" only.
	"russia": "ru", "moscow": "ru", "saint petersburg": "ru", "st petersburg": "ru",
	"kyiv": "ua", "kiev": "ua",
	"uzbekistan": "uz", "tashkent": "uz", "toshkent": "uz", "samarkand": "uz",
	"kazakhstan": "kz", "almaty": "kz", "astana": "kz", "nur-sultan": "kz",
	"kyrgyzstan": "kg", "bishkek": "kg",
	"tajikistan": "tj", "dushanbe": "tj",
	"turkmenistan": "tm", "ashgabat": "tm",
	"belarus": "by", "minsk": "by",
	"moldova": "md", "chisinau": "md",
	"armenia": "am", "yerevan": "am",
	"azerbaijan": "az", "baku": "az",
	"tbilisi": "ge",

	// Cyrillic, for the RU-segment ATS sources (sber, mts, alfabank, tbank, vk,
	// huntflow, …) whose location fields are in Russian. Seeded from the
	// high-frequency unresolved strings observed in production; grow by
	// observation. "россия"/"рф" are the country catch-all (the comma tokenizer
	// resolves "Самара, Россия" via the country token even when the city is
	// unknown). The "г "/"город " city-marker prefix is stripped before lookup
	// (stripCityPrefix), so only the bare city name is keyed.
	"россия": "ru", "рф": "ru",
	"москва": "ru", "санкт-петербург": "ru", "спб": "ru", "питер": "ru",
	"екатеринбург": "ru", "новосибирск": "ru", "нижний новгород": "ru",
	"казань": "ru", "самара": "ru", "краснодар": "ru", "ростов-на-дону": "ru",
	"воронеж": "ru", "уфа": "ru", "пермь": "ru", "челябинск": "ru",
	"волгоград": "ru", "красноярск": "ru", "омск": "ru", "тюмень": "ru",
	"саратов": "ru", "тольятти": "ru", "ижевск": "ru", "ульяновск": "ru",
	"барнаул": "ru", "владивосток": "ru", "хабаровск": "ru", "иркутск": "ru",
	"ярославль": "ru", "томск": "ru", "оренбург": "ru", "кемерово": "ru",
	"рязань": "ru", "набережные челны": "ru", "пенза": "ru", "липецк": "ru",
	"тула": "ru", "киров": "ru", "чебоксары": "ru", "калининград": "ru",
	"ставрополь": "ru", "сочи": "ru", "иваново": "ru", "брянск": "ru",
	"белгород": "ru", "сургут": "ru", "владимир": "ru", "архангельск": "ru",
	"калуга": "ru", "смоленск": "ru", "волжский": "ru", "курск": "ru",
	"орёл": "ru", "череповец": "ru", "вологда": "ru", "магнитогорск": "ru",
	"тамбов": "ru", "мурманск": "ru", "тверь": "ru", "новокузнецк": "ru",
	"астрахань": "ru", "великий новгород": "ru", "псков": "ru", "чита": "ru",
	"улан-удэ": "ru", "якутск": "ru", "норильск": "ru", "новороссийск": "ru",
	"таганрог": "ru", "сарапул": "ru", "майкоп": "ru", "подольск": "ru",
	"химки": "ru", "мытищи": "ru", "балашиха": "ru", "курган": "ru",
	"саранск": "ru", "йошкар-ола": "ru", "благовещенск": "ru", "кисловодск": "ru",
	"петропавловск-камчатский": "ru", "комсомольск-на-амуре": "ru",
	"новый уренгой": "ru",

	// CIS / Central Asia / Ukraine in Cyrillic, mirroring their Latin entries.
	"минск": "by", "беларусь": "by",
	"ташкент": "uz", "узбекистан": "uz",
	"алматы": "kz", "астана": "kz", "казахстан": "kz",
	"ереван": "am", "баку": "az", "бишкек": "kg",
	"киев": "ua", "київ": "ua",
}

// nameToRegion resolves macro-region names (and explicit open-anywhere markers)
// directly to a region code, for tokens that name an area rather than a country.
var nameToRegion = map[string]string{
	"europe": "eu", "eu": "eu",
	"emea": "emea", "eea": "eea",
	"apac": "apac", "asia": "apac", "asia pacific": "apac", "asia-pacific": "apac",
	"americas":      "americas",
	"north america": "north_america",
	"latam":         "latam", "latin america": "latam", "south america": "latam",
	"mena": "mena", "middle east": "mena",
	"africa": "africa",
	"cis":    "cis", "central asia": "central_asia",
	"anywhere": "global", "worldwide": "global", "global": "global", "remote anywhere": "global",
}
