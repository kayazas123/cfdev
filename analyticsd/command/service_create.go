package command

import (
	"encoding/json"
	"fmt"
	"gopkg.in/segmentio/analytics-go.v3"
	"log"
	"runtime"
	"time"
)

type ServiceCreate struct {
	CCClient        CloudControllerClient
	AnalyticsClient analytics.Client
	TimeStamp       time.Time
	UUID            string
	Version         string
	OSVersion       string
	Logger          *log.Logger
}

func (c *ServiceCreate) HandleResponse(body json.RawMessage) error {
	var metadata struct {
		Request struct {
			ServicePlanGuid string `json:"service_plan_guid"`
		}
	}

	json.Unmarshal(body, &metadata)

	var urlResp struct {
		Entity struct {
			ServiceURL string `json:"service_url"`
		}
	}

	path := "/v2/service_plans/" + metadata.Request.ServicePlanGuid
	err := c.CCClient.Fetch(path, nil, &urlResp)
	if err != nil {
		return fmt.Errorf("failed to make request to: %s: %s", path, err)
	}

	var labelResp struct {
		Entity struct {
			Label string
		}
	}

	path = urlResp.Entity.ServiceURL
	err = c.CCClient.Fetch(path, nil, &labelResp)
	if err != nil {
		return fmt.Errorf("failed to make request to: %s: %s", path, err)
	}

	if !serviceIsWhiteListed(labelResp.Entity.Label) {
		return nil
	}

	var properties = analytics.Properties{
		"service":        labelResp.Entity.Label,
		"os":             runtime.GOOS,
		"plugin_version": c.Version,
		"os_version":     c.OSVersion,
	}

	err = c.AnalyticsClient.Enqueue(analytics.Track{
		UserId:     c.UUID,
		Event:      "created service",
		Timestamp:  c.TimeStamp,
		Properties: properties,
	})

	if err != nil {
		return fmt.Errorf("failed to send analytics: %v", err)
	}

	return nil
}
