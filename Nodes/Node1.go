package main

import (
	proto "Mutual_Exclusion/gRPC"
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	proto.UnimplementedMutual_ExclusionServer
	Id      string
	port2Id map[string]proto.Mutual_ExclusionClient
	lTime   int64
	ReplyCount int
	InCS    bool
	Port	int
	deferredReplies []string
	RequestingCS bool
}

func main() {

	var port int
	var Id string
	fmt.Println("Enter port:")
	fmt.Scanln(&port)
	fmt.Println("Enter id:")
	fmt.Scanln(&Id)

	array := []int{8000, 8001, 8002}
	var arrayTwo [2]int = [2]int{}
	var num int = 0
	s := &Server{
		Id:      Id,
		Port:    port,
		lTime:   0,
		port2Id: make(map[string]proto.Mutual_ExclusionClient),
		deferredReplies: []string{},
		RequestingCS: false,
		ReplyCount: 0,
		InCS:    false,
	}

	go StartServer(s)
	time.Sleep(2 * time.Second)

	fmt.Println("Type Y when all 3 nodes are running...")
	fmt.Scanln()

	for _, v := range array {
		if !(v == port) {
			arrayTwo[num] = v
			num++
			fmt.Println(v)
		}
	}

if s.Id != "1" {
    conn1, err := grpc.Dial("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Could not connect to 1: %v", err)
    }
    s.port2Id["1"] = proto.NewMutual_ExclusionClient(conn1)
}
if s.Id != "2" {
    conn2, err := grpc.Dial("localhost:8001", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Could not connect to 2: %v", err)
    }
    s.port2Id["2"] = proto.NewMutual_ExclusionClient(conn2)
}
if s.Id != "3" {
    conn3, err := grpc.Dial("localhost:8002", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Could not connect to 3: %v", err)
    }
    s.port2Id["3"] = proto.NewMutual_ExclusionClient(conn3)
}

	time.Sleep(2 * time.Second)
	for {
		var message string

		fmt.Scanln(&message)

		if message == "request" {
			s.ReplyCount = 0
			s.RequestingCS = true
			for peerId, client := range s.port2Id {
        		if client != nil {
					requestString, err := client.Request(context.Background(), &proto.AdminRequest{ClientId: s.Id, LogicalTime: s.lTime,})
					if err != nil {
						log.Printf("Error sending request to %s: %v", peerId, err)
						continue
					}
					if requestString.Reply {
						s.ReplyCount++
					}
					s.lTime = max(s.lTime, requestString.LogicalTime) + 1
				}
			}

			if s.ReplyCount == 2 {
				log.Printf("Node %s entering critical section", Id)
				s.InCS = true
				time.Sleep(5 * time.Second)
				log.Printf("Node %s exiting critical section", Id)
				s.InCS = false
				s.RequestingCS = false
				for _, deferredId := range s.deferredReplies {
					client := s.port2Id[deferredId]
					client.SendReply(context.Background(), &proto.Reply{ClientId: s.Id, Reply: true, LogicalTime: s.lTime})
					log.Printf("Node %s sent deferred reply to Node %s", s.Id, deferredId)
					s.lTime++
				}
				s.deferredReplies = nil

			}
		}

		

		/*if message == "release" {
			s.lTime++
			//releaseString, _ = client.Release(context.Background(), &proto.AdminRelease{Id: Id, LogicalTime: lTime})
			client2.Release(context.Background(), &proto.AdminRelease{ClientId: s.Id, LogicalTime: s.lTime})
			client3.Release(context.Background(), &proto.AdminRelease{ClientId: s.Id, LogicalTime: s.lTime})
			s.ReplyCount = 0
			//log.Printf("Response from server (RELEASE): %v", releaseString)
			log.Printf("Node %s sent realease", s.Id)
		}*/

		
	}
}

func (s *Server) Request(ctx context.Context, req *proto.AdminRequest) (*proto.Reply, error) {
	log.Printf("Node %s received access request from Node %s", s.Id, req.ClientId)
	
    s.lTime = max(s.lTime, req.LogicalTime) + 1

	if s.InCS {
		log.Printf("Node %s is currently in critical section, delaying reply to Node %s", s.Id, req.ClientId)
		s.deferredReplies = append(s.deferredReplies, req.ClientId)
		return &proto.Reply{ClientId: s.Id, Reply: false, LogicalTime: s.lTime}, nil
	}

	if s.RequestingCS {
		if req.LogicalTime < s.lTime || (req.LogicalTime == s.lTime && req.ClientId < s.Id) {
			log.Printf("Node %s is requesting critical section, but will reply to Node %s", s.Id, req.ClientId)
		} else {
			log.Printf("Node %s is requesting critical section, delaying reply to Node %s", s.Id, req.ClientId)
			s.deferredReplies = append(s.deferredReplies, req.ClientId)
			return &proto.Reply{ClientId: s.Id, Reply: false, LogicalTime: s.lTime}, nil
		}
	}

	return &proto.Reply{ClientId: s.Id, Reply: true, LogicalTime: s.lTime}, nil
}


func StartServer(s *Server) {
    lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.Port))
    if err != nil {
        log.Fatalf("Failed to listen on port %d: %v", s.Port, err)
    }

    grpcServer := grpc.NewServer()
    proto.RegisterMutual_ExclusionServer(grpcServer, s)
    log.Printf("[%s] gRPC server started on port %d", s.Id, s.Port)

    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
