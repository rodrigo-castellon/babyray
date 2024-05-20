package config

import (
    "io/ioutil"
    "gopkg.in/yaml.v3"
    "log"
   // "os"
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

    // Construct the path to the configuration file. PROJECT_ROOT should somehow
    // be set prior to any Go code execution (i.e., in GitHub Actions workflow
    // before unit tests are run or in Dockerfile)
    rootPath := os.Getenv("PROJECT_ROOT")
    
    configFile := filepath.Join(rootPath, "config", "app_config.yaml")
   
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
