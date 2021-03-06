package main

import (
	"bytes"
	"context"
	"fmt"

	"flag"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/docker/docker/api/types/swarm"
)

var defaultName, encryptNet bool

//  Program assumes you are running inside an environment setup with a client bundle
func generateNetworkInfo(cli *client.Client) map[string]types.NetworkResource {

	networks, err := cli.NetworkList(
		context.Background(),
		types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}
	//call me lazy but I want to be able to refer easily to a network by its ID so create a mapping
	//between the id and the actual network resource
	theNetworks := make(map[string]types.NetworkResource)
	for _, network := range networks {
		theNetworks[network.ID] = network
	}

	return theNetworks
}

func buildStacks(theServices []swarm.Service, servicesbyID map[string]swarm.Service) map[string][]string {

	thisStacks := make(map[string][]string)
	for _, service := range theServices {
		labels := service.Spec.Labels
		name, ok := labels[convert.LabelNamespace]
		if !ok {
			continue
		} else {
			thisStacks[name] = append(thisStacks[name], service.ID)
		}
		servicesbyID[service.ID] = service
	}
	return thisStacks
}

func restartPolicyOut(thePolicy swarm.RestartPolicy) string {

	var outPut bytes.Buffer

	fmt.Fprintf(&outPut, "      restart_policy:\n")
	fmt.Fprintf(&outPut, "        condition: %s\n", thePolicy.Condition)
	if thePolicy.Delay == nil {
		fmt.Fprintf(&outPut, "        delay: 3s\n")
	} else {
		fmt.Fprintf(&outPut, "        delay: %s\n", thePolicy.Delay)
	}
	if thePolicy.MaxAttempts == nil {
		fmt.Fprintf(&outPut, "        max_attempts: 0\n")
	} else {
		fmt.Fprintf(&outPut, "        max_attempts: %s\n", strconv.FormatUint(uint64(*thePolicy.MaxAttempts), 10))
	}
	if thePolicy.Window == nil {
		fmt.Fprintf(&outPut, "        window: 0s\n")
	} else {
		fmt.Fprintf(&outPut, "        window: %s\n", thePolicy.Window)
	}
	return outPut.String()
}

func uporrollConfigOut(theConfig swarm.UpdateConfig) string {

	var outPut bytes.Buffer

	if theConfig.Parallelism != 0 {
		fmt.Fprintf(&outPut, "        parallelism: %d\n", theConfig.Parallelism)
	}
	if theConfig.Delay.String() != "" {
		fmt.Fprintf(&outPut, "        delay: %s\n", theConfig.Delay.String())
	}
	if theConfig.FailureAction != "" {
		fmt.Fprintf(&outPut, "        failure_action: %s\n", theConfig.FailureAction)
	}
	if theConfig.Monitor.String() != "" {
		fmt.Fprintf(&outPut, "        monitor: %s\n", theConfig.Monitor.String())
	}
	if theConfig.MaxFailureRatio != 0 {
		fmt.Fprintf(&outPut, "        max_failure_ratio %d\n", theConfig.MaxFailureRatio)
	}
	return outPut.String()

}

func constraintsOut(thePlacement swarm.Placement) string {

	var outPut bytes.Buffer

	fmt.Fprintf(&outPut, "      placement: \n")
	fmt.Fprintf(&outPut, "        constraints:\n")
	fmt.Fprintf(&outPut, "          - ")
	for loop, constraint := range thePlacement.Constraints {
		if loop == 0 {
			fmt.Fprintf(&outPut, "%s\n", constraint)
		} else {
			fmt.Fprintf(&outPut, "          - %s\n", constraint)
		}
	}
	return outPut.String()

}

func memandcpuOut(theInfo swarm.Resources) string {

	var outPut bytes.Buffer

	cpuInfo := float64(theInfo.NanoCPUs)
	memInfo := float64(theInfo.MemoryBytes)
	cpuInfo = cpuInfo / 1000000000
	memInfo = memInfo / 1048576
	fmt.Fprintf(&outPut, "          cpus: '%.2f'\n", cpuInfo)
	fmt.Fprintf(&outPut, "          memory: %.0fM\n", memInfo)
	return outPut.String()

}

func labelsOut(theLabels map[string]string, theIndent int) string {

	var outPut bytes.Buffer
	var spacer bytes.Buffer

	for counter := 0; counter < theIndent; counter++ {
		fmt.Fprintf(&spacer, " ")
	}
	fmt.Fprintf(&outPut, "%slabels:\n", &spacer)
	for key, value := range theLabels {
		fmt.Fprintf(&outPut, "%s  - ", &spacer)
		fmt.Fprintf(&outPut, "%s=%s\n", strings.Trim(key, " "), strings.Trim(value, " "))
	}
	return outPut.String()

}

func logInfoOut(logInfo swarm.Driver) string {

	var outPut bytes.Buffer

	if logInfo.Name == "" && (len(logInfo.Options) != 0) {
		fmt.Fprintf(&outPut, "    logging:\n")
		fmt.Fprintf(&outPut, "      options:\n")
		for key, value := range logInfo.Options {
			fmt.Fprintf(&outPut, "        %s: %s\n", key, value)
		}
	} else if logInfo.Name != "" &&
		len(logInfo.Options) != 0 {
		fmt.Fprintf(&outPut, "    logging:\n")
		fmt.Fprintf(&outPut, "      driver: %s\n", logInfo.Name)
		fmt.Fprintf(&outPut, "      options:\n")
		for key, value := range logInfo.Options {
			fmt.Fprintf(&outPut, "        %s: \"%s\"\n", key, value)
		}
	} else {
		fmt.Fprintf(&outPut, "    logging:\n")
		fmt.Fprintf(&outPut, "      driver: %s\n", logInfo.Name)
	}

	return outPut.String()
}

func processStack(stackName string, theStacks map[string][]string, serviceList map[string]swarm.Service, networkList map[string]types.NetworkResource) (outStr bytes.Buffer, myNetworks map[string]string) {

	fmt.Fprintf(&outStr, "version: '3.3'\n\n")
	fmt.Fprintf(&outStr, "services:\n")
	myNetworks = make(map[string]string)
	for _, serviceID := range theStacks[stackName] {
		if len(serviceList[serviceID].Spec.Networks) != 0 && len(serviceList[serviceID].Spec.Networks[0].Aliases) != 0 {
			fmt.Fprintf(&outStr, "  %s:\n", serviceList[serviceID].Spec.Networks[0].Aliases[0])
		} else {
			fmt.Fprintf(&outStr, "  %s:\n", serviceList[serviceID].Spec.Name)
		}
		//You have to have an image, I mean really and you at least need one replica
		fmt.Fprintf(&outStr, "    image: %s\n", serviceList[serviceID].Spec.TaskTemplate.ContainerSpec.Image)
		fmt.Fprintf(&outStr, "    deploy:\n")
		replicas := uint64(*serviceList[serviceID].Spec.Mode.Replicated.Replicas)
		fmt.Fprintf(&outStr, "      replicas: %s\n", strconv.FormatUint(replicas, 10))
		//if they have a restart policy, we need to deal with it
		if serviceList[serviceID].Spec.TaskTemplate.RestartPolicy != nil {
			fmt.Fprintf(&outStr, "%s", restartPolicyOut(*serviceList[serviceID].Spec.TaskTemplate.RestartPolicy))
		}
		if serviceList[serviceID].Spec.UpdateConfig != nil {
			fmt.Fprintf(&outStr, "      update_config:\n")
			fmt.Fprintf(&outStr, "%s", uporrollConfigOut(*serviceList[serviceID].Spec.UpdateConfig))
		}
		if serviceList[serviceID].Spec.RollbackConfig != nil {
			fmt.Fprintf(&outStr, "      rollback_config:\n")
			fmt.Fprintf(&outStr, "%s", uporrollConfigOut(*serviceList[serviceID].Spec.RollbackConfig))
		}
		if len(serviceList[serviceID].Spec.TaskTemplate.Placement.Constraints) != 0 {
			fmt.Fprintf(&outStr, "%s", constraintsOut(*serviceList[serviceID].Spec.TaskTemplate.Placement))
		}
		if serviceList[serviceID].Spec.TaskTemplate.Resources.Limits != nil || serviceList[serviceID].Spec.TaskTemplate.Resources.Reservations != nil {
			fmt.Fprintf(&outStr, "      resources: \n")
			if serviceList[serviceID].Spec.TaskTemplate.Resources.Limits != nil {
				fmt.Fprintf(&outStr, "        limits:\n")
				fmt.Fprintf(&outStr, "%s", memandcpuOut(*serviceList[serviceID].Spec.TaskTemplate.Resources.Limits))
			}
			if serviceList[serviceID].Spec.TaskTemplate.Resources.Reservations != nil {
				fmt.Fprintf(&outStr, "        reservations:\n")
				fmt.Fprintf(&outStr, "%s", memandcpuOut(*serviceList[serviceID].Spec.TaskTemplate.Resources.Reservations))
			}
		}
		//Ahhh yes the multiple locations of labels, this is for the deploy section
		if len(serviceList[serviceID].Spec.Annotations.Labels) != 0 {
			fmt.Fprintf(&outStr, "%s", labelsOut(serviceList[serviceID].Spec.Annotations.Labels, 6))
		}
		//And any published ports...
		if len(serviceList[serviceID].Endpoint.Spec.Ports) != 0 {
			fmt.Fprintf(&outStr, "    ports:\n")
			for _, port := range serviceList[serviceID].Endpoint.Spec.Ports {
				fmt.Fprintf(&outStr, "     - \n")
				fmt.Fprintf(&outStr, "\"%d:%d\"\n", port.PublishedPort, port.TargetPort)
			}
		}
		//yes you can have a service on no network, so we need to check that

		if len(serviceList[serviceID].Spec.TaskTemplate.Networks) != 0 {
			fmt.Fprintf(&outStr, "    networks:\n")
			for _, thisNetwork := range serviceList[serviceID].Spec.TaskTemplate.Networks {
				if networkList[thisNetwork.Target].Labels["com.docker.stack.namespace"] != "" {
					if defaultName {
						prefix := stackName + "_"
						theName := strings.TrimPrefix(networkList[thisNetwork.Target].Name, prefix)
						fmt.Fprintf(&outStr, "      - %s\n", theName)
					} else {
						fmt.Fprintf(&outStr, "      - %s\n", networkList[thisNetwork.Target].Name)
					}
				} else {
					fmt.Fprintf(&outStr, "      - %s\n", networkList[thisNetwork.Target].Name)
				}
				myNetworks[thisNetwork.Target] = networkList[thisNetwork.Target].Name
			}
		} else if len(serviceList[serviceID].Spec.Networks) != 0 {
			fmt.Fprintf(&outStr, "    networks:\n")
			for _, thisNetwork := range serviceList[serviceID].Spec.Networks {
				if networkList[thisNetwork.Target].Labels["com.docker.stack.namespace"] != "" {
					if defaultName {
						prefix := stackName + "_"
						theName := strings.TrimPrefix(networkList[thisNetwork.Target].Name, prefix)
						fmt.Fprintf(&outStr, "      - %s\n", theName)
					} else {
						fmt.Fprintf(&outStr, "      - %s\n", networkList[thisNetwork.Target].Name)
					}
				} else {
					fmt.Fprintf(&outStr, "      - %s\n", networkList[thisNetwork.Target].Name)
				}
				myNetworks[thisNetwork.Target] = networkList[thisNetwork.Target].Name
			}
		}
		//labels again, for the service specification
		if len(serviceList[serviceID].Spec.TaskTemplate.ContainerSpec.Labels) != 0 {
			fmt.Fprintf(&outStr, "%s", labelsOut(serviceList[serviceID].Spec.TaskTemplate.ContainerSpec.Labels, 4))
		}
		//Mounts for the service specification
		if len(serviceList[serviceID].Spec.TaskTemplate.ContainerSpec.Mounts) != 0 {
			fmt.Fprintf(&outStr, "    volumes:\n")
			for theMount := range serviceList[serviceID].Spec.TaskTemplate.ContainerSpec.Mounts {
				fmt.Fprintf(&outStr, "      - %s:%s\n", serviceList[serviceID].Spec.TaskTemplate.ContainerSpec.Mounts[theMount].Source, serviceList[serviceID].Spec.TaskTemplate.ContainerSpec.Mounts[theMount].Target)
			}
		}
		//and any environment variables
		if len(serviceList[serviceID].Spec.TaskTemplate.ContainerSpec.Env) > 0 {
			fmt.Fprintf(&outStr, "    environment:\n")
			for _, envVar := range serviceList[serviceID].Spec.TaskTemplate.ContainerSpec.Env {
				fmt.Fprintf(&outStr, "      - %s\n", envVar)
			}
		}
		//Log driver information gathered and provided
		if serviceList[serviceID].Spec.TaskTemplate.LogDriver != nil {
			fmt.Fprintf(&outStr, "%s", logInfoOut(*serviceList[serviceID].Spec.TaskTemplate.LogDriver))
		}
		fmt.Fprintf(&outStr, "\n")
	}
	return outStr, myNetworks
}

func processNetworkInfo(stackName string, outStr bytes.Buffer, networkList map[string]types.NetworkResource, foundNetworks map[string]string) bytes.Buffer {

	//So networks, need to dump those out, I am assuming in our implementation that if its not
	//external its overlay, and thats cause thats how we do it, call me lazy it works for us
	if len(foundNetworks) != 0 {
		fmt.Fprintf(&outStr, "networks:\n")
		for netID, netName := range foundNetworks {
			if networkList[netID].Labels["com.docker.stack.namespace"] == stackName {
				if defaultName {
					prefix := stackName + "_"
					theName := strings.TrimPrefix(netName, prefix)
					fmt.Fprintf(&outStr, "  %s:\n", theName)
				} else {
					fmt.Fprintf(&outStr, "  %s:\n", netName)
				}
				fmt.Fprintf(&outStr, "    driver: %s\n", networkList[netID].Driver)
				if len(networkList[netID].Options) != 0 {
					optString := ""
					matchEncrypted := false
					for name, value := range networkList[netID].Options {
						match1, _ := regexp.MatchString("vxlanid_list", name)
						match2, _ := regexp.MatchString("encrypted", name)
						matchEncrypted = matchEncrypted || match2
						if !match1 {
							if value == "" {
								optString = optString + fmt.Sprintf("        %s: \"\"\n", name)
							} else {
								optString = optString + fmt.Sprintf("        %s: %s\n", name, value)
							}
						}
					}
					if !matchEncrypted && encryptNet {
						optString = optString + fmt.Sprintf("        encrypted: \"\"\n")
					}
					if len(optString) > 0 {
						fmt.Fprintf(&outStr, "    driver_opts:\n")
						fmt.Fprintf(&outStr, "%s", optString)
					}
				}
				if len(networkList[netID].Labels["com.docker.ucp.access.label"]) != 0 {
					fmt.Fprintf(&outStr, "    labels:\n")
					fmt.Fprintf(&outStr, "       - com.docker.ucp.access.label = %s\n", networkList[netID].Labels["com.docker.ucp.access.label"])
				}
			} else {
				fmt.Fprintf(&outStr, "  %s:\n", netName)
				fmt.Fprintf(&outStr, "    external: true\n")
			}
		}
	}
	return outStr
}

func main() {
	//setup flags, right now just one
	stackPtr := flag.String("stack", "*", "a string of the pattern to match for stacks")
	unamePtr := flag.Bool("unname", false, "do not set network name, i.e. use default")
	encryptPtr := flag.Bool("encrypt", false, "force networks to be created encrypted")
	helpPtr := flag.Bool("help", false, "display help message")
	flag.Parse()

	defaultName = *unamePtr
	encryptNet = *encryptPtr

	if *helpPtr {
		flag.PrintDefaults()
		os.Exit(0)
	}
	//setup client environment

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	//grab all the services
	services, err := cli.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}
	matched := false
	all := false
	wildcard := strings.Compare(*stackPtr, "*")
	if wildcard == 0 {
		all = true
	} else {
		all = false
	}

	allTheNetworks := generateNetworkInfo(cli)
	allTheServices := make(map[string]swarm.Service)
	stacks := buildStacks(services, allTheServices)
	//setup stuff to do matching for the stackname and what we actually want for output
	//so now we go through the stacks to find the one(s) that match the request and then do some work

	for stackname, _ := range stacks {
		matched, _ = regexp.MatchString(*stackPtr, stackname)
		if matched || all {
			fmt.Printf("//******** Stackname is - %s *******************//\n", stackname)
			fmt.Println("//********************************************************************************************//")
			stackCompose, myNetworks := processStack(stackname, stacks, allTheServices, allTheNetworks)
			stackCompose = processNetworkInfo(stackname, stackCompose, allTheNetworks, myNetworks)
			fmt.Println(stackCompose.String())
			fmt.Println("//********************************************************************************************//")
		}
	}
}
