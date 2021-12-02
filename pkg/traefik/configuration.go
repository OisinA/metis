package traefik

type Configuration struct {
	HTTP HttpConfig
}

type Router struct {
	Rule    string `json:"rule"`
	Service string `json:"service"`
}

type HttpConfigs struct {
	Configs map[string]HttpConfig
}

type HttpConfig struct {
	Routers  map[string]Router
	Services map[string]Service
}

type Service struct {
	LoadBalancer LoadBalancer
}

type LoadBalancer struct {
	Servers []URL
}

type URL struct {
	URL string `json:"url"`
}

func (c Configuration) ToMap() map[string]interface{} {
	config := map[string]interface{}{}
	config["http"] = map[string]interface{}{}
	routers := map[string]interface{}{}

	for key, router := range c.HTTP.Routers {
		rtr := map[string]interface{}{}
		rtr["rule"] = router.Rule
		rtr["service"] = router.Service
		routers[key] = rtr
	}

	config["http"].(map[string]interface{})["routers"] = routers

	services := map[string]interface{}{}

	for key, service := range c.HTTP.Services {
		srv := map[string]interface{}{}
		loadBalancer := map[string]interface{}{}
		loadBalancer["servers"] = []map[string]string{}
		for _, url := range service.LoadBalancer.Servers {
			loadBalancer["servers"] = append(loadBalancer["servers"].([]map[string]string), map[string]string{
				"url": url.URL,
			})
		}

		srv["loadBalancer"] = loadBalancer
		services[key] = srv
	}

	config["http"].(map[string]interface{})["services"] = services

	return config
}
