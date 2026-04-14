package environment

type Config struct {
	MongoURI   string
	KafkaTopic string
}

func InitializeConfig() Config {
	return Config{
		MongoURI:   "mongodb://localhost:27017",
		KafkaTopic: "user-events",
	}
}
