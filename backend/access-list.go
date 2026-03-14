package main

import (
	_ "embed"
	"log"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// accessListYamlContent keeps the access catalog inside the backend binary so route lookups
// do not depend on the current working directory or external files at runtime.
//
//go:embed access_list.yml
var accessListYamlContent []byte

type AccessListYaml struct {
	AccessGroups []struct {
		ID   int32  `yaml:"id"`
		Name string `yaml:"name"`
	} `yaml:"access_groups"`
	AccessList []struct {
		ID             int32  `yaml:"id"`
		Name           string `yaml:"name"`
		Group          int32  `yaml:"group"`
		Levels         int32  `yaml:"levels"`
		FrontendRoutes string `yaml:"frontend_routes"`
		BackendRoutes  string `yaml:"backend_routes"`
		BackendAPIs    string `yaml:"backend_apis"`
	} `yaml:"access_list"`
}

type AccessInfo struct {
	ID   int32
	Name string
}

var (
	routeAccessMap map[string]AccessInfo
	accessMapOnce  sync.Once
)

func getRouteAccessInfo(route string) (AccessInfo, bool) {
	accessMapOnce.Do(func() {
		routeAccessMap = make(map[string]AccessInfo)
		var parsedAccessList AccessListYaml
		if err := yaml.Unmarshal(accessListYamlContent, &parsedAccessList); err != nil {
			log.Printf("[access-list] failed to unmarshal embedded access_list.yml: %v", err)
			return
		}

		// Split comma-separated backend routes once so access checks stay O(1) at request time.
		for _, accessListEntry := range parsedAccessList.AccessList {
			backendRoutes := strings.Split(accessListEntry.BackendRoutes, ",")
			for _, backendRoute := range backendRoutes {
				trimmedBackendRoute := strings.TrimSpace(backendRoute)
				if trimmedBackendRoute == "" {
					continue
				}

				routeAccessMap[trimmedBackendRoute] = AccessInfo{
					ID:   accessListEntry.ID,
					Name: accessListEntry.Name,
				}
			}
		}
	})

	info, ok := routeAccessMap[route]
	return info, ok
}
