package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/cli/cli/compose/convert"
	//"github.com/docker/docker/api"
	"github.com/docker/docker/api/types/swarm"
	"regexp"
	"flag"
	"strings"
	"strconv"
)


func main() {
	cli, err := client.NewEnvClient()
	//theServices := make(map[string][]map[string]string)
	if err != nil {
		panic(err)
	}

    	services, err := cli.ServiceList(context.Background(),types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}
    	networks, err := cli.NetworkList(
		context.Background(),
		types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}
	theNetworks := make(map[string]types.NetworkResource)
	for _, network := range networks {
		theNetworks[network.ID] = network
	}
	stackPtr := flag.String("stack", "*", "a string")
	flag.Parse()
	stacks := make(map[string][]string)
	theServices := make(map[string]swarm.Service)
	for _, service := range services {
		labels := service.Spec.Labels
		name, ok := labels[convert.LabelNamespace]
		if !ok {
			continue
		} else {
		    stacks[name] = append(stacks[name],service.ID)
		}
		theServices[service.ID] = service
	}
	matched := false
	all := false
	wildcard := strings.Compare(*stackPtr,"*")
	if wildcard == 0 {
		all = true
	} else {
	    all = false
    }    
	fmt.Println("Request is -",*stackPtr)
	fmt.Println()
	for stackname, services := range stacks {
		matched, _ = regexp.MatchString(*stackPtr,stackname)
		if matched || all {
			fmt.Println("Stackname is - ", stackname)
			fmt.Printf("version: '3'\n\n")
			fmt.Println("services:")
			myNetworks := make(map[string]string)
			for _,serviceID := range services {
				if len(theServices[serviceID].Spec.Networks) != 0 && len(theServices[serviceID].Spec.Networks[0].Aliases) != 0 {
					fmt.Printf("  %s:\n",theServices[serviceID].Spec.Networks[0].Aliases[0])
				} else {
					fmt.Printf("  %s:\n", theServices[serviceID].Spec.Name)
				}	
				fmt.Println("    image: ",theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Image)
				fmt.Println("    deploy:")
				replicas := uint64(*theServices[serviceID].Spec.Mode.Replicated.Replicas)
				fmt.Println("      replicas: ",strconv.FormatUint(replicas,10))
				if theServices[serviceID].Spec.TaskTemplate.RestartPolicy != nil {
					fmt.Println("      restart_policy:")
					fmt.Println("        condition: ",theServices[serviceID].Spec.TaskTemplate.RestartPolicy.Condition)
					fmt.Println("        delay: ",theServices[serviceID].Spec.TaskTemplate.RestartPolicy.Delay)
					fmt.Println("        max_attempts: ",strconv.FormatUint(uint64(*theServices[serviceID].Spec.TaskTemplate.RestartPolicy.MaxAttempts),10))
					fmt.Println("        window: ",theServices[serviceID].Spec.TaskTemplate.RestartPolicy.Window)
				}
				if len(theServices[serviceID].Spec.TaskTemplate.Placement.Constraints) != 0 {
					fmt.Println("      placement: ")
					fmt.Println("        constraints:")
					fmt.Print("          - ")
					for loop, constraint := range theServices[serviceID].Spec.TaskTemplate.Placement.Constraints {
						if loop == 0 {
							fmt.Println(constraint)
						} else {
							fmt.Println("          -",constraint)
						}	
					}
				}
				if len(theServices[serviceID].Spec.Annotations.Labels) != 0 {
					fmt.Println("      labels:")
					for key, value := range theServices[serviceID].Spec.Annotations.Labels {
						fmt.Print("        - ")
						fmt.Printf("%s=%s\n",strings.Trim(key," "),strings.Trim(value," "))
					}
				}
				if len(theServices[serviceID].Endpoint.Spec.Ports) != 0  {
					fmt.Println("    ports:")
					for _, port := range theServices[serviceID].Endpoint.Spec.Ports {
						fmt.Print("     - ")
						fmt.Printf("\"%d:%d\"\n",port.PublishedPort,port.TargetPort)
					}	
				}
				if len(theServices[serviceID].Spec.Networks) != 0 {
					fmt.Println("    networks:")
					for _, thisNetwork := range theServices[serviceID].Spec.Networks {
						fmt.Println("      -",theNetworks[thisNetwork.Target].Name)
						myNetworks[thisNetwork.Target] = theNetworks[thisNetwork.Target].Name
					}
				}
				if len(theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Labels) != 0 {
					fmt.Println("    labels:")
					for key, value := range theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Labels {
						fmt.Printf("      - %s=%s\n",key,value)
					}
				}	
				if len(theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Env) > 0 {
					fmt.Println("    environment:")
					for _, envVar := range theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Env {
						fmt.Printf("      - %s\n",envVar)
					}
					
				}
				if theServices[serviceID].Spec.TaskTemplate.LogDriver != nil {
					if theServices[serviceID].Spec.TaskTemplate.LogDriver.Name == "" &&
					   len(theServices[serviceID].Spec.TaskTemplate.LogDriver.Options) != 0   {
						fmt.Println("    logging:")
						fmt.Println("      options:")
						for key, value := range theServices[serviceID].Spec.TaskTemplate.LogDriver.Options  {
							fmt.Printf("        %s: %s\n",key,value)
						}
					} else if theServices[serviceID].Spec.TaskTemplate.LogDriver.Name != "" && 
					   len(theServices[serviceID].Spec.TaskTemplate.LogDriver.Options) != 0   {
						fmt.Println("    logging:")
						fmt.Println("      driver:", theServices[serviceID].Spec.TaskTemplate.LogDriver.Name)
						fmt.Println("      options:")
						for key, value := range theServices[serviceID].Spec.TaskTemplate.LogDriver.Options  {
							fmt.Printf("        %s: %s\n",key,value)
						}
					} else {
						fmt.Println("    logging:")
						fmt.Println("      driver:", theServices[serviceID].Spec.TaskTemplate.LogDriver.Name)			
					}
				}
				fmt.Println()	
			}
			if len(myNetworks) != 0 {
				fmt.Println("networks:")
				for netID,netName := range myNetworks {
					fmt.Printf("  %s:\n",netName)
					if theNetworks[netID].Labels["com.docker.stack.namespace"] != "" {
						fmt.Println("    driver: overlay")
					} else {
						fmt.Println("    external: true")
					}
				}
			}		
		}	
	}
}