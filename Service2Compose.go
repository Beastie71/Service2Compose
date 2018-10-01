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
	"os"
)
//  Program assumes you are running inside an environment setup with a client bundle

func main() {
	//setup flags, right now just one	
	stackPtr := flag.String("stack", "*", "a string of the pattern to match for stacks")
	unamePtr := flag.Bool("unname", false, "do not set network name, i.e. use default")
	encryptPtr := flag.Bool("encrypt", false, "force networks to be created encrypted")
	helpPtr := flag.Bool("help", false, "display help message")
	flag.Parse()
	
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
    	services, err := cli.ServiceList(context.Background(),types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}
	//grab all the networks
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
	//Grab all the stacks, unfortunately stacks is really represented as a label on a service
	//so we need to go through all the services and and get all the values for that label AND
	//while we are at it we should keep track of all the services for that stack
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
	//setup stuff to do matching for the stackname and what we actually want for output
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
	//so now we go through the stacks to find the one(s) that match the request and then do some work
	for stackname, services := range stacks {
		matched, _ = regexp.MatchString(*stackPtr,stackname)
		if matched || all {
			fmt.Println("Stackname is - ", stackname)
			fmt.Println()
			//And here we go actually dumping out the compose
			fmt.Printf("version: '3.3'\n\n")
			fmt.Println("services:")
			myNetworks := make(map[string]string)
			for _,serviceID := range services {
				//So maybe just particular to how we are doing composes, but we are using the alias and
				//we aren't using multiple aliases so really I am JUST setting up the initial one
				//unless of course somehow thats not been set, then I just put the service name in there
				if len(theServices[serviceID].Spec.Networks) != 0 && len(theServices[serviceID].Spec.Networks[0].Aliases) != 0 {
					fmt.Printf("  %s:\n",theServices[serviceID].Spec.Networks[0].Aliases[0])
				} else {
					fmt.Printf("  %s:\n", theServices[serviceID].Spec.Name)
				}	
				//You have to have an image, I mean really and you at least need one replica
				fmt.Println("    image: ",theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Image)
				fmt.Println("    deploy:")
				replicas := uint64(*theServices[serviceID].Spec.Mode.Replicated.Replicas)
				fmt.Println("      replicas: ",strconv.FormatUint(replicas,10))
				//if they have a restart policy, we need to deal with it
				if theServices[serviceID].Spec.TaskTemplate.RestartPolicy != nil {
					fmt.Println("      restart_policy:")
					fmt.Println("        condition: ",theServices[serviceID].Spec.TaskTemplate.RestartPolicy.Condition)
					if theServices[serviceID].Spec.TaskTemplate.RestartPolicy.Delay == nil {
						fmt.Println("        delay: 3s")
					} else {
						fmt.Println("        delay: ",theServices[serviceID].Spec.TaskTemplate.RestartPolicy.Delay)
					}
					if theServices[serviceID].Spec.TaskTemplate.RestartPolicy.MaxAttempts == nil {
						fmt.Println("        max_attempts: 0")
					} else {
						fmt.Println("        max_attempts: ",strconv.FormatUint(uint64(*theServices[serviceID].Spec.TaskTemplate.RestartPolicy.MaxAttempts),10))
					}	
					if theServices[serviceID].Spec.TaskTemplate.RestartPolicy.Window == nil {
						fmt.Println("        window: 0s")
					} else {
						fmt.Println("        window: ",theServices[serviceID].Spec.TaskTemplate.RestartPolicy.Window)
					}	
				}
				if ( theServices[serviceID].Spec.UpdateConfig != nil) {
					fmt.Println("      update_config:")
				    if ( theServices[serviceID].Spec.UpdateConfig.Parallelism != 0 ) {
				    	fmt.Printf("        parallelism: %d\n",theServices[serviceID].Spec.UpdateConfig.Parallelism)
				    }
				    if ( theServices[serviceID].Spec.UpdateConfig.Delay.String() != "" ) {
				    	fmt.Printf("        delay: %s\n",theServices[serviceID].Spec.UpdateConfig.Delay.String())
				    }
   				    if ( theServices[serviceID].Spec.UpdateConfig.FailureAction != "" ) {
				    	fmt.Printf("        failure_action: %s\n",theServices[serviceID].Spec.UpdateConfig.FailureAction)
				    }
				    if ( theServices[serviceID].Spec.UpdateConfig.Monitor.String() != "" ) {
				    	fmt.Printf("        monitor: %s\n",theServices[serviceID].Spec.UpdateConfig.Monitor.String())
				    }
				    if ( theServices[serviceID].Spec.UpdateConfig.MaxFailureRatio != 0 ) {
				    	fmt.Printf("        max_failure_ratio %d\n",theServices[serviceID].Spec.UpdateConfig.MaxFailureRatio)
				    }
//				    if ( theServices[serviceID].Spec.UpdateConfig.Order != "" ) {
//				    	fmt.Printf("        order: %s\n",theServices[serviceID].Spec.UpdateConfig.Order)
//				    }
				}
//				
//				
//				   order is not supported till v 3.3 of the compose file format, so commenting out for now.
//				
//				
				if ( theServices[serviceID].Spec.RollbackConfig != nil) {
					fmt.Println("      rollback_config:")
				    if ( theServices[serviceID].Spec.UpdateConfig.Parallelism != 0 ) {
				    	fmt.Printf("        parallelism: %d\n",theServices[serviceID].Spec.UpdateConfig.Parallelism)
				    }
				    if ( theServices[serviceID].Spec.UpdateConfig.Delay.String() != "" ) {
				    	fmt.Printf("        delay: %s\n",theServices[serviceID].Spec.UpdateConfig.Delay.String())
				    }
   				    if ( theServices[serviceID].Spec.UpdateConfig.FailureAction != "" ) {
				    	fmt.Printf("        failure_action: %s\n",theServices[serviceID].Spec.UpdateConfig.FailureAction)
				    }
				    if ( theServices[serviceID].Spec.UpdateConfig.Monitor.String() != "" ) {
				    	fmt.Printf("        monitor: %s\n",theServices[serviceID].Spec.UpdateConfig.Monitor.String())
				    }
				    if ( theServices[serviceID].Spec.UpdateConfig.MaxFailureRatio != 0 ) {
				    	fmt.Printf("        max_failure_ratio %d\n",theServices[serviceID].Spec.UpdateConfig.MaxFailureRatio)
				    }
//				    if ( theServices[serviceID].Spec.UpdateConfig.Order != "" ) {
//				    	fmt.Printf("        order: %s\n",theServices[serviceID].Spec.UpdateConfig.Order)
//				    }
				}

				//if they have constraints we need to deal with that
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
				if ( theServices[serviceID].Spec.TaskTemplate.Resources.Limits != nil || theServices[serviceID].Spec.TaskTemplate.Resources.Reservations != nil) {
					fmt.Println("      resources: ")
					if ( theServices[serviceID].Spec.TaskTemplate.Resources.Limits != nil ) {
						cpuInfo := float64(theServices[serviceID].Spec.TaskTemplate.Resources.Limits.NanoCPUs)
						memInfo := float64(theServices[serviceID].Spec.TaskTemplate.Resources.Limits.MemoryBytes)
						cpuInfo = cpuInfo / 1000000000
						memInfo = memInfo / 1048576
						fmt.Println("        limits:")
						fmt.Printf("          cpus: '%.2f'\n",cpuInfo)
						fmt.Printf("          memory: %.0fM\n",memInfo)
					}
					if ( theServices[serviceID].Spec.TaskTemplate.Resources.Reservations != nil ) {
						cpuInfo := float64(theServices[serviceID].Spec.TaskTemplate.Resources.Limits.NanoCPUs)
						memInfo := float64(theServices[serviceID].Spec.TaskTemplate.Resources.Limits.MemoryBytes)
						cpuInfo = cpuInfo / 1000000000
						memInfo = memInfo / 1048576
						fmt.Println("        reservations:")
						fmt.Printf("          cpus: '%.2f'\n",cpuInfo)
						fmt.Printf("          memory: %.0fM\n",memInfo)
					}
				}
				//Ahhh yes the multiple locations of labels, this is for the deploy section
				if len(theServices[serviceID].Spec.Annotations.Labels) != 0 {
					fmt.Println("      labels:")
					for key, value := range theServices[serviceID].Spec.Annotations.Labels {
						fmt.Print("        - ")
						fmt.Printf("%s=%s\n",strings.Trim(key," "),strings.Trim(value," "))
					}
				}
				//And any published ports...
				if len(theServices[serviceID].Endpoint.Spec.Ports) != 0  {
					fmt.Println("    ports:")
					for _, port := range theServices[serviceID].Endpoint.Spec.Ports {
						fmt.Print("     - ")
						fmt.Printf("\"%d:%d\"\n",port.PublishedPort,port.TargetPort)
					}	
				}
				//yes you can have a service on no network, so we need to check that
				//fmt.Println(theServices[serviceID].Spec.TaskTemplate.Networks)
				if len(theServices[serviceID].Spec.TaskTemplate.Networks) != 0 {
					fmt.Println("    networks:")
					for _, thisNetwork := range theServices[serviceID].Spec.TaskTemplate.Networks {
						if theNetworks[thisNetwork.Target].Labels["com.docker.stack.namespace"] != "" {
							if *unamePtr {
								prefix := stackname + "_"
								theName := strings.TrimPrefix(theNetworks[thisNetwork.Target].Name, prefix)
								fmt.Println("      -",theName)
							} else {
								fmt.Println("      -",theNetworks[thisNetwork.Target].Name)						
							}
						} else {
							fmt.Println("      -",theNetworks[thisNetwork.Target].Name)
						}
						myNetworks[thisNetwork.Target] = theNetworks[thisNetwork.Target].Name
					}
				} else if len(theServices[serviceID].Spec.Networks) != 0 {
					fmt.Println("    networks:")
					for _, thisNetwork := range theServices[serviceID].Spec.Networks {
						if theNetworks[thisNetwork.Target].Labels["com.docker.stack.namespace"] != "" {
							if *unamePtr {
								prefix := stackname + "_"
								theName := strings.TrimPrefix(theNetworks[thisNetwork.Target].Name, prefix)
								fmt.Println("      -",theName)
							} else {
								fmt.Println("      -",theNetworks[thisNetwork.Target].Name)						
							}
						} else {
							fmt.Println("      -",theNetworks[thisNetwork.Target].Name)
						}
						myNetworks[thisNetwork.Target] = theNetworks[thisNetwork.Target].Name
					}
				}						
					 				//labels again, for the service specification
				if len(theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Labels) != 0 {
					fmt.Println("    labels:")
					for key, value := range theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Labels {
						fmt.Printf("      - %s=%s\n",key,value)
					}
				}	
				//Mounts for the service specification
				if len(theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Mounts) != 0 {
					fmt.Println("    volumes:")
					for theMount := range theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Mounts {
						fmt.Printf("      - %s:%s\n",theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Mounts[theMount].Source,theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Mounts[theMount].Target)
					}
				}	

				//and any environment variables
				if len(theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Env) > 0 {
					fmt.Println("    environment:")
					for _, envVar := range theServices[serviceID].Spec.TaskTemplate.ContainerSpec.Env {
						fmt.Printf("      - %s\n",envVar)
					}
					
				}
				//Log driver information gathered and provided
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
							fmt.Printf("        %s: \"%s\"\n",key,value)
						}
					} else {
						fmt.Println("    logging:")
						fmt.Println("      driver:", theServices[serviceID].Spec.TaskTemplate.LogDriver.Name)			
					}
				}
				fmt.Println()	
			}
			//So networks, need to dump those out, I am assuming in our implementation that if its not
			//external its overlay, and thats cause thats how we do it, call me lazy it works for us
			if len(myNetworks) != 0 {
				fmt.Println("networks:")
				for netID,netName := range myNetworks {
					if theNetworks[netID].Labels["com.docker.stack.namespace"] == *stackPtr {
						if *unamePtr {
							prefix := stackname + "_"
							theName := strings.TrimPrefix(netName, prefix)
							fmt.Printf("  %s:\n",theName)
						} else {
								fmt.Printf("  %s:\n",netName)					
						}
						fmt.Println("    driver:",theNetworks[netID].Driver)
						if(len(theNetworks[netID].Options) != 0 ) {
							optString := ""
							matchEncrypted := false
							for name, value := range theNetworks[netID].Options {
								match1, _ := regexp.MatchString("vxlanid_list",name)
								match2, _ := regexp.MatchString("encrypted",name)
								matchEncrypted = matchEncrypted || match2
								if ( ! match1 ) {
									if ( value == "" ) {
										optString = optString + fmt.Sprintf("        %s: \"\"\n",name)
									} else {
									    optString = optString + fmt.Sprintf("        %s: %s\n",name,value)
									}
								}	
							}
							if ( ! matchEncrypted && *encryptPtr ) {
								optString = optString + fmt.Sprintf("        encrypted: \"\"\n")
							}
							if (len(optString) > 0 ) {
								fmt.Println("    driver_opts:")
								fmt.Printf("%s",optString)
							}
						}
						if ( len(theNetworks[netID].Labels["com.docker.ucp.access.label"]) != 0 ) {
						  fmt.Println("    labels:")
						  fmt.Println("       com.docker.ucp.access.label:",theNetworks[netID].Labels["com.docker.ucp.access.label"])
						}  
					} else {
						fmt.Printf("  %s:\n",netName)
						fmt.Println("    external: true")
					}
				}
			}		
		}	
	}
}