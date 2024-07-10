package main

import (
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/util/sets"
)

type configuration struct {
	ConfigItems accessConfig `json:"access,omitempty"`
}

func (c *configuration) Validate() error {
	return c.ConfigItems.validate()
}

func (c *configuration) SetDefault() {
	// do nothing
}

type accessConfig struct {
	// Plugins is a map of repositories (eg "k/k") to lists of plugin names.
	RepoPlugins map[string][]string `json:"repo_plugins,omitempty"`

	// Plugins is a list available plugins.
	Plugins []pluginConfig `json:"plugins,omitempty"`
}

type pluginConfig struct {
	// Name of the plugin.
	Name string `json:"name" required:"true"`

	// Endpoint is the location of the plugin.
	Endpoint string `json:"endpoint" required:"true"`

	// Events are the events that this plugin can handle and should be forward to it.
	// If no events are specified, everything is sent.
	Events []string `json:"events,omitempty"`
}

func (a *accessConfig) validate() error {
	var botSet = sets.String{}
	for i := range a.Plugins {
		if err := a.Plugins[i].validate(); err != nil {
			return err
		}
		botSet.Insert(a.Plugins[i].Name)
	}

	var e []string
	for _, item := range a.RepoPlugins {
		for _, value := range item {
			if !botSet.Has(value) {
				e = append(e, value)
			}
		}
	}

	if len(e) > 0 {
		return fmt.Errorf("config.yaml existed unknown plugins: %v", e)
	}

	return nil
}

func (c *configuration) GetEndpoints(org, repo, eventType string) (ans []string) {

	if c.ConfigItems.RepoPlugins == nil {
		return []string{}
	}

	var robotNames []string
	endpoint, ok := c.ConfigItems.RepoPlugins[org]
	if ok {
		robotNames = append(robotNames, endpoint...)
	}

	endpoint, ok = c.ConfigItems.RepoPlugins[org+"/"+repo]
	if ok {
		robotNames = append(robotNames, endpoint...)
	}

	if len(c.ConfigItems.Plugins) != 0 && len(robotNames) != 0 {
		ans = matchEndpoint(&c.ConfigItems.Plugins, eventType, robotNames...)
	}

	return
}

func matchEndpoint(m *[]pluginConfig, event string, robotNames ...string) (ans []string) {
	for _, val := range robotNames {
		for _, value := range *m {
			if value.Name == val {
				sort.Strings(value.Events)
				idx := sort.SearchStrings(value.Events, event)
				if idx < len(value.Events) && value.Events[idx] == event {
					ans = append(ans, value.Endpoint)
				}
			}
		}
	}

	return
}

func (p *pluginConfig) validate() error {
	if p.Name == "" {
		return fmt.Errorf("missing name")
	}

	if p.Endpoint == "" {
		return fmt.Errorf("missing endpoint")
	}

	// p.Endpoint unchecked
	return nil
}
