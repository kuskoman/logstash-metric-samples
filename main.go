package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/client"
	apiClient "github.com/kuskoman/logstash-metric-samples/internal/client"
	"github.com/kuskoman/logstash-metric-samples/internal/config"
	"github.com/kuskoman/logstash-metric-samples/internal/containers"
)

const outputDir = "output"

func main() {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	versions, err := config.GetVersions()
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(versions))

	for _, version := range versions {
		go func(version string) {
			defer wg.Done()
			err := scrapeVersion(dockerCli, version)
			if err != nil {
				log.Printf("Error scraping version %s: %s", version, err)
			}
		}(version)
	}

	wg.Wait()
	log.Println("Done")
}

func scrapeVersion(dockerCli *client.Client, version string) error {
	timeout := 25 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	containerID, port, err := containers.SpawnLogstashContainer(ctx, dockerCli, version)
	if err != nil {
		return err
	}

	defer func() {
		removeContainerCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		containers.RemoveContainer(removeContainerCtx, dockerCli, containerID)
	}()

	err = containers.WaitForContainerToBeReady(ctx, port)
	if err != nil {
		return err
	}

	logstashUrl := fmt.Sprintf("http://localhost:%d", port)
	nodeStatsUrl := fmt.Sprintf("%s/_node/stats", logstashUrl)
	nodeInfoUrl := fmt.Sprintf("%s/_node/", logstashUrl)

	log.Printf("Getting node stats from %s", nodeStatsUrl)
	nodeStats, err := apiClient.GetJSONFromAPI(ctx, nodeStatsUrl)
	if err != nil {
		return err
	}

	log.Printf("Getting node info from %s", nodeInfoUrl)
	nodeInfo, err := apiClient.GetJSONFromAPI(ctx, nodeInfoUrl)
	if err != nil {
		return err
	}

	outputDir := fmt.Sprintf("%s/%s", outputDir, version)
	log.Printf("Ensuring output directory %s exists", outputDir)
	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	nodeStatsFile := fmt.Sprintf("%s/node-stats.json", outputDir)
	nodeInfoFile := fmt.Sprintf("%s/node-info.json", outputDir)

	log.Printf("Writing node stats to %s", nodeStatsFile)
	err = apiClient.WriteJSONToFile(nodeStats, nodeStatsFile)
	if err != nil {
		return err
	}

	log.Printf("Writing node info to %s", nodeInfoFile)
	err = apiClient.WriteJSONToFile(nodeInfo, nodeInfoFile)
	if err != nil {
		return err
	}

	log.Printf("Done scraping version %s", version)
	return nil
}
