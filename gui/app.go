package main

// App struct
type App struct {
	config *Config
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		config: LoadConfig(),
	}
}
