package nameserver

import (
	"github.com/fsouza/go-dockerclient"
)

func StartUpdater(apiPath string, zone Zone) error {
	client, err := docker.NewClient(apiPath)
	if err != nil {
		Error.Fatalf("Unable to connect to Docker API on %s: %s", apiPath, err)
	}

	events := make(chan *docker.APIEvents)
	client.AddEventListener(events)

	Info.Printf("Using Docker API on %s", apiPath)

	go func() {
		for event := range events {
			handleEvent(zone, event, client)
		}
	}()
	return nil
}

func handleEvent(zone Zone, event *docker.APIEvents, client *docker.Client) error {
	switch event.Status {
	case "die":
		id := event.ID
		Info.Printf("Container %s down. Removing records", id)
		zone.DeleteRecordsFor(id)
	}
	return nil
}
