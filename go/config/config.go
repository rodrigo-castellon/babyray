package config

import (
    "io/ioutil"
    "gopkg.in/yaml.v3"
    "log"
)

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

func LoadConfig() Config {
    var config Config
    yamlFile, err := ioutil.ReadFile("config/app_config.yaml")
    if err != nil {
        log.Fatalf("error: %v", err)
    }
    err = yaml.Unmarshal(yamlFile, &config)
    if err != nil {
        log.Fatalf("Unmarshal: %v", err)
    }
    return config
}

