package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Food-to-Share/bridge/config"
	"github.com/Food-to-Share/bridge/database"
	flag "maunium.net/go/mauflag"
	log "maunium.net/go/maulogger/v2"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/id"
)

var configPath = flag.MakeFull("c", "config", "The path to your config file.", "config.yaml").String()
var registrationPath = flag.MakeFull("r", "registration", "The path where to save the appservice registration.", "registration.yaml").String()
var generateRegistration = flag.MakeFull("g", "generate-registration", "Generate registration and quit.", "false").Bool()
var wantHelp, _ = flag.MakeHelpFlag()

func (bridge *Bridge) GenerateRegistration() {
	reg, err := bridge.Config.NewRegistration()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to generate registration:", err)
		os.Exit(20)
	}

	err = reg.Save(*registrationPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save registration:", err)
		os.Exit(21)
	}

	err = bridge.Config.Save(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save config:", err)
		os.Exit(22)
	}
	fmt.Println("Registration generated. Add the path to the registration to your Synapse config restart it, then start the bridge.")
	os.Exit(0)
}

type Bridge struct {
	AS             *appservice.AppService
	EventProcessor *appservice.EventProcessor
	MatrixHandler  *MatrixHandler
	Config         *config.Config
	DB             *database.Database
	Log            log.Logger
	StateStore     *database.SQLStateStore
	Provisioning   *ProvisioningAPI
	Bot            *appservice.IntentAPI

	usersByMXID         map[id.UserID]*User
	usersByJID          map[string]*User
	usersLock           sync.Mutex
	managementRooms     map[id.RoomID]*User
	managementRoomsLock sync.Mutex
	portalsByMXID       map[id.RoomID]*Portal
	portalsByJID        map[database.PortalKey]*Portal
	portalsLock         sync.Mutex
	puppets             map[string]*Puppet
	puppetsLock         sync.Mutex
}

func NewBridge() *Bridge {
	bridge := &Bridge{
		usersByMXID: make(map[id.UserID]*User),
		usersByJID:  make(map[string]*User),
		// spaceRooms:          make(map[id.RoomID]*User),
		managementRooms: make(map[id.RoomID]*User),
		portalsByMXID:   make(map[id.RoomID]*Portal),
		portalsByJID:    make(map[database.PortalKey]*Portal),
		puppets:         make(map[string]*Puppet),
		// puppetsByCustomMXID: make(map[id.UserID]*Puppet),
	}
	var err error
	bridge.Config, err = config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(10)
	}
	return bridge
}

func (bridge *Bridge) ensureConnection() {
	for {
		resp, err := bridge.Bot.Whoami()
		if err != nil {
			if errors.Is(err, mautrix.MUnknownToken) {
				bridge.Log.Fatalln("The as_token was not accepted. Is the registration file installed in your homeserver correctly?")
				os.Exit(16)
			} else if errors.Is(err, mautrix.MExclusive) {
				bridge.Log.Fatalln("The as_token was accepted, but the /register request was not. Are the homeserver domain and username template in the config correct, and do they match the values in the registration?")
				os.Exit(16)
			}
			bridge.Log.Errorfln("Failed to connect to homeserver: %v. Retrying in 10 seconds...", err)
			time.Sleep(10 * time.Second)
		} else if resp.UserID != bridge.Bot.UserID {
			bridge.Log.Fatalln("Unexpected user ID in whoami call: got %s, expected %s", resp.UserID, bridge.Bot.UserID)
			os.Exit(17)
		} else {
			break
		}
	}
}

func (bridge *Bridge) Init() {
	var err error
	bridge.AS, err = bridge.Config.MakeAppService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to initialize AppService:", err)
		os.Exit(11)
	}
	bridge.AS.Init()
	bridge.Bot = bridge.AS.BotIntent()
	bridge.Log = log.Create()
	bridge.Config.Logging.Configure(bridge.Log)
	log.DefaultLogger = bridge.Log.(*log.BasicLogger)

	err = log.OpenFile()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to open log file:", err)
		os.Exit(12)
	}
	bridge.AS.Log = log.Sub("Matrix")

	bridge.Log.Debugln("Initializing database")
	bridge.DB, err = database.New(bridge.Config.AppService.Database.Type, bridge.Config.AppService.Database.URI)
	if err != nil {
		bridge.Log.Fatalln("Failed to initialize database:", err)
		os.Exit(14)
	}

	bridge.Log.Debugln("Initializing state store")
	bridge.StateStore = database.NewSQLStateStore(bridge.DB)
	if err != nil {
		bridge.Log.Fatalln("Failed to load state store:", err)
		os.Exit(13)
	}
	bridge.AS.StateStore = bridge.StateStore

	bridge.Provisioning = &ProvisioningAPI{bridge: bridge}

	bridge.Log.Debugln("Initializing Matrix event processor")
	bridge.EventProcessor = appservice.NewEventProcessor(bridge.AS)
	bridge.Log.Debugln("Initializing Matrix event handler")
	bridge.MatrixHandler = NewMatrixHandler(bridge)
}

func (bridge *Bridge) Start() {
	err := bridge.DB.CreateTables()
	if err != nil {
		bridge.Log.Fatalfln("Failed to create database tables:", err)
		os.Exit(15)
	}
	bridge.Log.Debugln("Checking connection to homeserver")
	bridge.ensureConnection()
	if bridge.Provisioning != nil {
		bridge.Log.Debugln("Initializing provisioning API")
		bridge.Provisioning.Init()
	}
	bridge.Log.Debugln("Starting application service HTTP server")
	go bridge.AS.Start()
	bridge.Log.Debugln("Starting event processor")
	go bridge.EventProcessor.Start()
	go bridge.UpdateBotProfile()
	go bridge.StartUsers()
}

func (bridge *Bridge) UpdateBotProfile() {
	bridge.Log.Debugln("Updating bot profile")
	botConfig := bridge.Config.AppService.Bot

	var err error
	if botConfig.Displayname == "remove" {
		err = bridge.Bot.SetDisplayName("")
	}
	if err != nil {
		bridge.Log.Warnln("Failed to update bot displayname:", err)
	}
}

func (bridge *Bridge) StartUsers() {
	// for _, user := range bridge.GetAllUsers() {
	// 	user.log.Debug("Chegou aqui")
	// 	user.log.Debug("User MXID: " + user.MXID + ", User JID" + id.UserID(user.JID))
	// }
}

func (bridge *Bridge) Stop() {
	bridge.AS.Stop()
	bridge.EventProcessor.Stop()
}

func (bridge *Bridge) Main() {
	if *generateRegistration {
		bridge.GenerateRegistration()
		return
	}

	bridge.Init()
	bridge.Log.Infoln("Bridge initialization complete, starting...")
	bridge.Start()
	bridge.Log.Infoln("Bridge started!")

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	bridge.Log.Infoln("Interrupt received, stopping...")
	bridge.Stop()
	bridge.Log.Infoln("Bridge stopped.")
	os.Exit(0)
}

func main() {
	flag.SetHelpTitles("foodToShare - A foodToShare puppeting bridge.", "[-h] [-c <path>] [-r <path>] [-g]")
	err := flag.Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.PrintHelp()
		os.Exit(1)
	} else if *wantHelp {
		flag.PrintHelp()
		os.Exit(0)
	}

	NewBridge().Main()
}
