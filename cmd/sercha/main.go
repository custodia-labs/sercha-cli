package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/custodia-labs/sercha-cli/cgo/xapian"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/ai"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/auth"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/config/file"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/storage/sqlite"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/cli"
	"github.com/custodia-labs/sercha-cli/internal/connectors"
	"github.com/custodia-labs/sercha-cli/internal/core/services"
	"github.com/custodia-labs/sercha-cli/internal/normalisers"
	"github.com/custodia-labs/sercha-cli/internal/postprocessors"
)

var version = "dev"

func main() {
	os.Exit(run())
}

//nolint:funlen // main initialisation requires sequential setup of all dependencies
func run() int {
	cli.SetVersion(version)

	// Create unified SQLite store for all metadata persistence
	sqliteStore, err := sqlite.NewStore("")
	if err != nil {
		log.Printf("failed to create SQLite store: %v", err)
		return 1
	}
	defer sqliteStore.Close()

	// Get individual store interfaces from unified store
	sourceStore := sqliteStore.SourceStore()
	syncStore := sqliteStore.SyncStateStore()
	docStore := sqliteStore.DocumentStore()
	exclusionStore := sqliteStore.ExclusionStore()
	schedulerStore := sqliteStore.SchedulerStore()
	authProviderStore := sqliteStore.AuthProviderStore()
	credentialsStore := sqliteStore.CredentialsStore()

	// Create config store and settings service EARLY (needed for AI adapter creation)
	configStore, err := file.NewConfigStore("")
	if err != nil {
		log.Printf("failed to create config store: %v", err)
		return 1
	}
	aiConfigValidator := ai.NewConfigValidator()
	settingsSvc := services.NewSettingsService(configStore, aiConfigValidator)

	// Get current settings to determine which adapters to create
	settings, err := settingsSvc.Get()
	if err != nil {
		log.Printf("failed to get settings: %v", err)
		return 1
	}

	// Create Xapian search engine (always needed for keyword search)
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("failed to get home directory: %v", err)
		return 1
	}
	xapianPath := filepath.Join(home, ".sercha", "data", "xapian")
	if err := os.MkdirAll(xapianPath, 0700); err != nil {
		log.Printf("failed to create Xapian directory: %v", err)
		return 1
	}
	searchEngine, err := xapian.New(xapianPath)
	if err != nil {
		log.Printf("failed to create Xapian search engine: %v", err)
		return 1
	}
	defer searchEngine.Close()

	// Initialise AI services with auto-fallback on failure
	vectorPath := filepath.Join(home, ".sercha", "data", "vectors")
	if err := os.MkdirAll(vectorPath, 0700); err != nil {
		log.Printf("failed to create vector directory: %v", err)
		return 1
	}

	aiResult, err := ai.InitialiseServices(settings, vectorPath)
	if err != nil {
		log.Printf("fatal error initialising AI: %v", err)
		return 1
	}
	defer aiResult.Close()

	// Log warnings and notify user about fallback
	if aiResult.FellBack {
		log.Println("Warning: Running in text-only mode due to AI configuration issues:")
		for _, w := range aiResult.Warnings {
			log.Printf("  - %s", w)
		}
		log.Println("Run 'sercha settings wizard' to configure AI features.")
	}

	// Provider registry is created after connector registry (see below)

	// Create auth services (AuthProvider/Credentials architecture)
	authProviderSvc := services.NewAuthProviderService(authProviderStore, sourceStore)
	credentialsSvc := services.NewCredentialsService(credentialsStore)

	// Create TokenProviderFactory for connector authentication
	tokenProviderFactory := auth.NewFactory(credentialsStore, authProviderStore)

	// Create connector and normaliser registries
	connectorFactory := connectors.NewFactory(tokenProviderFactory)
	normaliserRegistry := normalisers.NewRegistry()

	// Create PostProcessor pipeline from configuration
	pipelineCfg := settingsSvc.GetPipelineConfig()
	processorRegistry := postprocessors.NewRegistry()
	postprocessors.RegisterDefaults(processorRegistry)

	pipeline := postprocessors.NewPipeline()
	for _, name := range pipelineCfg.Processors {
		cfg := pipelineCfg.GetProcessorConfig(name)
		processor, err := processorRegistry.Build(name, cfg)
		if err != nil {
			log.Printf("failed to build processor %s: %v", name, err)
			return 1
		}
		pipeline.Add(processor)
	}

	// Create core services with AI dependencies
	searchSvc := services.NewSearchService(
		docStore, searchEngine, aiResult.VectorIndex,
		aiResult.EmbeddingService, aiResult.LLMService,
	)
	// Set optional stores for SourceName enrichment in search results
	searchSvc.SetSourceStore(sourceStore)
	searchSvc.SetCredentialsStore(credentialsStore)

	sourceSvc := services.NewSourceService(sourceStore, syncStore, docStore)

	// Create connector registry (needed before sourceSvc.SetConnectorRegistry)
	connectorRegistry := services.NewConnectorRegistry(connectorFactory)
	sourceSvc.SetConnectorRegistry(connectorRegistry)

	// Create provider registry (depends on connectorRegistry and connectorFactory)
	providerRegistry := services.NewProviderRegistry(connectorRegistry, connectorFactory)

	syncSvc := services.NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore, connectorFactory, normaliserRegistry,
		pipeline, searchEngine, aiResult.VectorIndex, aiResult.EmbeddingService,
	)
	resultActionSvc := services.NewResultActionService(sourceStore, connectorRegistry)
	documentSvc := services.NewDocumentService(docStore, sourceStore, exclusionStore, connectorRegistry)

	// Create scheduler (started only by TUI command which is long-running)
	schedulerCfg := settingsSvc.GetSchedulerConfig()
	scheduler := services.NewScheduler(
		schedulerCfg,
		schedulerStore,
		syncSvc,
	)

	// Inject services into CLI commands
	cli.SetServices(&cli.Services{
		Search:            searchSvc,
		Source:            sourceSvc,
		Sync:              syncSvc,
		Document:          documentSvc,
		ConnectorRegistry: connectorRegistry,
		ProviderRegistry:  providerRegistry,
		Settings:          settingsSvc,
		AuthProvider:      authProviderSvc,
		Credentials:       credentialsSvc,
	})

	// Inject services into TUI command (including scheduler for background tasks)
	cli.SetTUIConfig(&cli.TUIConfig{
		SearchService:       searchSvc,
		SourceService:       sourceSvc,
		SyncOrchestrator:    syncSvc,
		ResultActionService: resultActionSvc,
		DocumentService:     documentSvc,
		ConnectorRegistry:   connectorRegistry,
		ProviderRegistry:    providerRegistry,
		SettingsService:     settingsSvc,
		CredentialsService:  credentialsSvc,
		AuthProviderService: authProviderSvc,
		Scheduler:           scheduler,
		SchedulerConfig:     schedulerCfg,
	})

	if err := cli.Execute(); err != nil {
		return 1
	}
	return 0
}
