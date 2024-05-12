package config

import (
    "io/ioutil"
    "gopkg.in/yaml.v3"
    "log"
    "os"
    "path/filepath"
)

var cfg *Config

type Config struct {
    NumWorkerNodes int `yaml:"num_worker_nodes"`
    ClusterSize    int `yaml:"cluster_size"`
    NodeIDs        struct {
        GCS             int `yaml:"gcs"`
        GlobalScheduler int `yaml:"global_scheduler"`
        Ourself         int `yaml:"ourself"`
    } `yaml:"node_ids"`
    Ports struct {
        LocalObjectStore   int `yaml:"local_object_store"`
        LocalScheduler     int `yaml:"local_scheduler"`
        LocalWorkerStart   int `yaml:"local_worker_start"`
        GCSFunctionTable   int `yaml:"gcs_function_table"`
        GCSObjectTable     int `yaml:"gcs_object_table"`
        GlobalScheduler    int `yaml:"global_scheduler"`
    } `yaml:"ports"`
    DNS struct {
        NodePrefix string `yaml:"node_prefix"`
    } `yaml:"dns"`
}

func LoadConfig() *Config {
    var config Config

    // Get the working directory of the executable. Key assumption here is that
    // the executable is located at go/cmd/*/[executable]. Otherwise this will
    // break.
    cwd, err := os.Getwd()
    if err != nil {
        log.Fatalf("Failed to get current working directory: %v", err)
    }

    // Construct the path to the configuration file
    rootPath := os.Getenv("PROJECT_ROOT")
    // _ = rootPath
    log.Printf("%v", rootPath)
    log.Printf("%v", cwd)
    log.Printf("ROOT IS: %v", filepath.Join(rootPath, "config", "app_config.yaml"))
    configFile := filepath.Join(cwd, "..", "..", "..", "config", "app_config.yaml")
    // configFile := filepath.Join(rootPath, "config", "app_config.yaml")

    yamlFile, err := ioutil.ReadFile(configFile)
    if err != nil {
        log.Fatalf("error: %v", err)
    }
    err = yaml.Unmarshal(yamlFile, &config)
    if err != nil {
        log.Fatalf("Unmarshal: %v", err)
    }
    return &config
}

func init() {
    cfg = LoadConfig()
}

func GetConfig() *Config {
    return cfg
}
