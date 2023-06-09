package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"

	pb "live/pb"
)

var (
	Port = flag.String("Port", "50000", "Port")
	List sync.Map
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)
	flag.Parse()
}

type userServiceServer struct {
	pb.UnimplementedUserServiceServer
}

func (s *userServiceServer) GetUser(ctx context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	// 在這裡實現取得使用者的邏輯
	// 可以根據 req.UID 執行相應的操作
	// 然後回傳 UserResponse 物件

	if data, ok := List.Load(req.UID); ok {
		return &pb.UserResponse{Info: data.(*pb.UserInfo)}, nil
	} else {
		return nil, errors.New("User Not Found")
	}
}

func (s *userServiceServer) CreateUser(ctx context.Context, req *pb.UserInfo) (*pb.CreateUserResponse, error) {
	// 在這裡實現建立使用者的邏輯
	// 可以根據 req 中的資訊進行使用者建立的相應操作
	// 然後回傳 CreateUserResponse 物件

	if _, ok := List.LoadOrStore(req.UID, req); ok {
		return nil, errors.New("Data already exists")

	} else {
		return &pb.CreateUserResponse{
			Result: pb.Status_Succ,
		}, nil
	}
}

func (s *userServiceServer) GetList(ctx context.Context, _ *pb.Empty) (*pb.ListResponse, error) {
	list := make([]*pb.UserInfo, 0)
	List.Range(func(key, value any) bool {
		list = append(list, value.(*pb.UserInfo))
		return true
	})
	return &pb.ListResponse{List: list}, nil
}

func main() {

	defer func() {
		if r := recover(); r != nil {
			log.Println("server panic: ", r)
		}
	}()

	go func() {

		// 建立 gRPC 伺服器
		server := grpc.NewServer()

		// 註冊 UserServiceServer
		userService := &userServiceServer{}
		pb.RegisterUserServiceServer(server, userService)

		// 監聽指定的網路位址
		lis, err := net.Listen("tcp", ":"+*Port)
		if err != nil {
			log.Fatalf("無法監聽: %v", err)
		}

		// 開始接受連線並處理請求
		if err := server.Serve(lis); err != nil {
			log.Fatalf("無法啟動伺服器: %v", err)
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for ch := range signalCh {
			switch ch {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				log.Println("Program Exit...", ch)

				time.Sleep(1 * time.Second)

				os.Exit(0)
			default:
				log.Println("signal.Notify default")

			}
		}
	}()

	for {
		log.Print(">")
		reader := bufio.NewReader(os.Stdin)
		text, _, _ := reader.ReadLine()

		switch strings.ToLower(string(text)) {

		case "test":
			Test()
			log.Println("gRPC Run..")
		case "quit":

			time.Sleep(1 * time.Second)

			return

		}
	}
}

func Test() {

	conn, err := grpc.Dial(":"+*Port, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("伺服器無法連線: %v", err)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	//Create User 1
	userInfoRequest := &pb.UserInfo{
		UID:   "1",
		Name:  "Alice",
		Token: "abc123",
	}
	createUserResponse, err := client.CreateUser(context.Background(), userInfoRequest)
	if err != nil {
		log.Printf("CreateUser 1錯誤: %v\n", err)
	} else {

		log.Printf("status:%v\n", createUserResponse.Result)

	}

	//GetUser 1
	userRequest := &pb.UserRequest{UID: userInfoRequest.UID}
	userResponse, err := client.GetUser(context.Background(), userRequest)
	if err != nil {
		log.Printf("取得User: %v\n", err)
	}

	userInfo := userResponse.GetInfo()
	log.Printf("UID=%s, Name=%s, Token=%s\n", userInfo.UID, userInfo.Name, userInfo.Token)

	//Create User 2
	userInfoRequest2 := &pb.UserInfo{
		UID:   "2",
		Name:  "Alice",
		Token: "abc123",
	}
	createUserResponse, err = client.CreateUser(context.Background(), userInfoRequest2)
	if err != nil {
		log.Printf("CreateUser 2錯誤: %v\n", err)
	} else {
		log.Printf("status:%v\n", createUserResponse.Result)
	}

	//Get Not Exist User
	userRequest2 := &pb.UserRequest{UID: "123"}
	userResponse, err = client.GetUser(context.Background(), userRequest2)
	if err != nil {
		log.Printf("取得User: %v\n", err)
	}

	//Get List
	listResponse, err := client.GetList(context.Background(), &pb.Empty{})
	if err != nil {
		log.Printf("GetList 錯誤: %v\n", err)
	} else {
		bytes, _ := json.Marshal(listResponse.List)
		log.Println(string(bytes))

	}

}
