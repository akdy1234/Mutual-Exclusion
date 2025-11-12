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
	tempTime int64
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
		tempTime: 0,
	}
	go StartServer(s)
	time.Sleep(2 * time.Second)

	fmt.Println("Enter any key when all 3 nodes are running...")
	fmt.Scanln()

	for _, v := range array {
		if !(v == port) {
			arrayTwo[num] = v
			num++
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
			s.lTime++
			s.tempTime = s.lTime
			for peerId, client := range s.port2Id {
        		if client != nil {
					requestString, err := client.Request(context.Background(), &proto.AdminRequest{ClientId: s.Id, LogicalTime: s.tempTime,})
					if err != nil {
						log.Printf("Error sending request to %s: %v", peerId, err)
						continue
					}
					if requestString.Reply {
						s.ReplyCount++
					}
					log.Printf("updating logical time.")
					log.Printf("before: %d", s.lTime)
					s.lTime = max(s.lTime, requestString.LogicalTime) + 1
					log.Printf("after: %d", s.lTime)
				}
				
			}
		
			for s.ReplyCount < 2 {
				time.Sleep(5 * time.Second)
				log.Printf("Waiting for reply... (%d replies)", s.ReplyCount)
				
			}
			log.Printf("%d replys recieved", s.ReplyCount)
			
			if s.ReplyCount == 2 {
				log.Printf("Node %s entering critical section", Id)
				s.InCS = true
			}
		}
		if message == "release" {
			
			log.Printf("Node %s exiting critical section", Id)
			s.InCS = false
			s.RequestingCS = false

			for _, deferredId := range s.deferredReplies {
					client := s.port2Id[deferredId]
					client.SendReply(context.Background(), &proto.Reply{ClientId: s.Id, Reply: true, LogicalTime: s.lTime})
					log.Printf("Node %s sent deferred reply to Node %s", s.Id, deferredId)
					log.Printf("updating logical time.")
					log.Printf("before: %d", s.lTime)
					s.lTime++
					log.Printf("after: %d", s.lTime)
				}
				s.deferredReplies = nil
			
		}

		
	}
}

func (s *Server) Request(ctx context.Context, req *proto.AdminRequest) (*proto.Reply, error) {
	log.Printf("Node %s received access request from Node %s", s.Id, req.ClientId)

	log.Printf("updating logical time.")
	log.Printf("before: %d", s.lTime)
    s.lTime = max(s.lTime, req.LogicalTime) + 1
	log.Printf("after: %d", s.lTime)

	if s.InCS {
		log.Printf("Node %s is currently in critical section, delaying reply to Node %s", s.Id, req.ClientId)
		s.deferredReplies = append(s.deferredReplies, req.ClientId)
		return &proto.Reply{ClientId: s.Id, Reply: false, LogicalTime: s.lTime}, nil
	}
	if s.RequestingCS {
		if req.LogicalTime < s.tempTime || (req.LogicalTime == s.tempTime && req.ClientId < s.Id) {
			log.Printf("Node %s is requesting critical section, but will reply to Node %s", s.Id, req.ClientId)
		} else {
			log.Printf("Node %s is requesting critical section, delaying reply to Node %s", s.Id, req.ClientId)
			s.deferredReplies = append(s.deferredReplies, req.ClientId)
			return &proto.Reply{ClientId: s.Id, Reply: false, LogicalTime: s.lTime}, nil
		}
	}

	return &proto.Reply{ClientId: s.Id, Reply: true, LogicalTime: s.lTime}, nil
}

func (s *Server) SendReply(ctx context.Context, rep *proto.Reply) (*proto.Empty, error) {
	log.Printf("Node %s received reply from Node %s", s.Id, rep.ClientId)
	
	log.Printf("updating logical time.")
	log.Printf("before: %d", s.lTime)
    s.lTime = max(s.lTime, rep.LogicalTime) + 1
	log.Printf("after: %d", s.lTime)
	s.ReplyCount++
	return &proto.Empty{}, nil
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
