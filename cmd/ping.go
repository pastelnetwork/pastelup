package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"

	pb "github.com/pastelnetwork/gonode/proto/healthcheck"
	"google.golang.org/grpc"
)

var (
	superNodeIP   string
	superNodePort int
)

func setupPingCommand() *cli.Command {
	config := configs.InitConfig()

	pingCommand := cli.NewCommand("ping")
	pingCommand.SetUsage("To check the health of service")
	addLogFlags(pingCommand, config)

	pingSuperCommandFlags := []*cli.Flag{
		cli.NewFlag("ip", &superNodeIP).
			SetUsage(red("Required, ip address of service")).SetRequired(),
		cli.NewFlag("port", &superNodePort).
			SetUsage(red("Required, port of service")).SetRequired(),
	}

	pingSuperCommand := cli.NewCommand("supernode")
	pingSuperCommand.SetUsage(cyan("check supernode healthcheck"))
	pingSuperCommand.AddFlags(pingSuperCommandFlags...)
	pingSuperCommand.SetActionFunc(func(ctx context.Context, _ []string) error {
		ctx, err := configureLogging(ctx, "ping ", config)
		if err != nil {
			//Logger doesn't exist
			return fmt.Errorf("failed to configure logging option - %v", err)
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sys.RegisterInterruptHandler(cancel, func() {
			log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
			os.Exit(0)
		})

		log.WithContext(ctx).Info("Started")
		if err = runPingSuperNode(ctx, config); err != nil {
			return err
		}
		log.WithContext(ctx).Info("Finished successfully!")
		return nil
	})

	pingCommand.AddSubcommands(pingSuperCommand)
	return pingCommand
}

func runPingSuperNode(ctx context.Context, _ *configs.Config) error {

	if len(superNodeIP) == 0 {
		return fmt.Errorf("--ip <IP address> - Required, ip address of service")
	}

	if superNodePort == 0 {
		return fmt.Errorf("--port <port value> - Required, port of service")
	}

	serverAddr := fmt.Sprintf("%s:%d", superNodeIP, superNodePort)
	// Prepare the client
	log.WithContext(ctx).Info("Connecting to supernode service...")
	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(subCtx, serverAddr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to connect to supernode service ")
		return err
	}
	defer conn.Close()
	client := pb.NewHealthCheckClient(conn)
	log.WithContext(ctx).Info("Connected successful to supernode service ")

	// Send ping request
	log.WithContext(ctx).Info("Sending ping command...")

	subCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	res, err := client.Ping(subCtx, &pb.PingRequest{Msg: "hello"})
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to send ping command ")
		return err
	}
	log.WithContext(ctx).Infof("Ping sucessfully, received reply: %s", res.Reply)

	return nil
}
