package config

import "os"

func DefaultStore() *Store { return NewStore(os.Getenv("APPDATA")) }
