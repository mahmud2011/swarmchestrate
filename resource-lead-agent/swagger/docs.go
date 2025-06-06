// Package swagger Code generated by swaggo/swag. DO NOT EDIT
package swagger

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/swarmchestrate/api/v1/applications": {
            "get": {
                "description": "Returns all applications currently registered",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "applications"
                ],
                "summary": "List all applications",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/domain.Application"
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            },
            "post": {
                "description": "Creates a new application from uploaded YAML service files",
                "consumes": [
                    "multipart/form-data"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "applications"
                ],
                "summary": "Create an application",
                "parameters": [
                    {
                        "type": "file",
                        "description": "Application YAML",
                        "name": "application",
                        "in": "formData"
                    },
                    {
                        "type": "file",
                        "description": "Cloud service YAML",
                        "name": "cloud_svc",
                        "in": "formData"
                    },
                    {
                        "type": "file",
                        "description": "Fog service YAML",
                        "name": "fog_svc",
                        "in": "formData"
                    },
                    {
                        "type": "file",
                        "description": "Edge service YAML",
                        "name": "edge_svc",
                        "in": "formData"
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/domain.Application"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/swarmchestrate/api/v1/applications/{id}": {
            "get": {
                "description": "Returns application details by UUID",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "applications"
                ],
                "summary": "Get application by ID",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Application ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/domain.Application"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            },
            "delete": {
                "description": "Deletes an application by UUID",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "applications"
                ],
                "summary": "Delete an application",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Application ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            },
            "patch": {
                "description": "Updates an existing application using uploaded YAML service files",
                "consumes": [
                    "multipart/form-data"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "applications"
                ],
                "summary": "Update an application",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Application ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "file",
                        "description": "Application YAML",
                        "name": "application",
                        "in": "formData"
                    },
                    {
                        "type": "file",
                        "description": "Cloud service YAML",
                        "name": "cloud_svc",
                        "in": "formData"
                    },
                    {
                        "type": "file",
                        "description": "Fog service YAML",
                        "name": "fog_svc",
                        "in": "formData"
                    },
                    {
                        "type": "file",
                        "description": "Edge service YAML",
                        "name": "edge_svc",
                        "in": "formData"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/domain.Application"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/swarmchestrate/api/v1/clusters": {
            "get": {
                "description": "Returns all clusters, optionally filtered by type",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "clusters"
                ],
                "summary": "List clusters",
                "parameters": [
                    {
                        "enum": [
                            "cloud",
                            "fog",
                            "edge"
                        ],
                        "type": "string",
                        "description": "Cluster Type",
                        "name": "type",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/domain.Cluster"
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            },
            "post": {
                "description": "Accepts cluster configuration and registers it in the system",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "clusters"
                ],
                "summary": "Create a new cluster",
                "parameters": [
                    {
                        "description": "Cluster request payload",
                        "name": "cluster",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/dto.AddClusterReq"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/domain.Cluster"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/swarmchestrate/api/v1/clusters/config": {
            "get": {
                "description": "Used by worker clusters to fetch current configuration (e.g., IPs)",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "clusters"
                ],
                "summary": "Get cluster configuration for worker clusters",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {}
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/swarmchestrate/api/v1/clusters/{cluster-id}/applications/scheduled": {
            "get": {
                "description": "Returns all apps scheduled for the given cluster",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "applications"
                ],
                "summary": "List scheduled applications for a cluster",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Cluster ID",
                        "name": "cluster-id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/dto.ScheduledApplication"
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/swarmchestrate/api/v1/clusters/{cluster-id}/applications/{application-id}": {
            "get": {
                "description": "Returns a single scheduled app by cluster ID and app ID",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "applications"
                ],
                "summary": "Get a specific scheduled application for a cluster",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Cluster ID",
                        "name": "cluster-id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Application ID",
                        "name": "application-id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/dto.ScheduledApplication"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found"
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/swarmchestrate/api/v1/clusters/{cluster-id}/applications/{application-id}/status": {
            "post": {
                "description": "Used by a cluster to report the current state of a scheduled app",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "applications"
                ],
                "summary": "Report application status from a worker cluster",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Cluster ID",
                        "name": "cluster-id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Application ID",
                        "name": "application-id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Status payload",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/dto.ClusterApplicationStatus"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found"
                    },
                    "406": {
                        "description": "Not Acceptable"
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/swarmchestrate/api/v1/clusters/{id}": {
            "get": {
                "description": "Returns a cluster by UUID if it exists",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "clusters"
                ],
                "summary": "Get cluster by ID",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Cluster ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/domain.Cluster"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            },
            "delete": {
                "description": "Deletes a cluster by UUID",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "clusters"
                ],
                "summary": "Delete a cluster",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Cluster ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            },
            "patch": {
                "description": "Updates an existing cluster's configuration",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "clusters"
                ],
                "summary": "Update a cluster",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Cluster ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Cluster update payload",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/dto.UpdateClusterReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/dto.UpdateClusterResp"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/swarmchestrate/api/v1/clusters/{id}/nodes": {
            "put": {
                "description": "Adds or updates nodes in the specified cluster",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "nodes"
                ],
                "summary": "Upsert cluster nodes",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Cluster ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Node upsert payload",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/dto.UpsertNode"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/domain.Node"
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "domain.Application": {
            "type": "object",
            "properties": {
                "cloud_svc": {
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                },
                "cloud_svc_cluster": {
                    "type": "string"
                },
                "cloud_svc_heartbeat": {
                    "type": "string"
                },
                "cloud_svc_nodes": {
                    "type": "string"
                },
                "cloud_svc_status": {
                    "type": "string"
                },
                "cloud_svc_version": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "edge_svc": {
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                },
                "edge_svc_cluster": {
                    "type": "string"
                },
                "edge_svc_heartbeat": {
                    "type": "string"
                },
                "edge_svc_nodes": {
                    "type": "string"
                },
                "edge_svc_status": {
                    "type": "string"
                },
                "edge_svc_version": {
                    "type": "string"
                },
                "fog_svc": {
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                },
                "fog_svc_cluster": {
                    "type": "string"
                },
                "fog_svc_heartbeat": {
                    "type": "string"
                },
                "fog_svc_nodes": {
                    "type": "string"
                },
                "fog_svc_status": {
                    "type": "string"
                },
                "fog_svc_version": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "labels": {
                    "$ref": "#/definitions/domain.Labels"
                },
                "name": {
                    "type": "string"
                },
                "qos": {
                    "$ref": "#/definitions/domain.QoS"
                },
                "updated_at": {
                    "type": "string"
                }
            }
        },
        "domain.ApplicationState": {
            "type": "string",
            "enum": [
                "progressing",
                "healthy",
                "failed",
                "NULL"
            ],
            "x-enum-varnames": [
                "StateProgressing",
                "StateHealthy",
                "StateFailed",
                "StateNULL"
            ]
        },
        "domain.Cluster": {
            "type": "object",
            "properties": {
                "created_at": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "ip": {
                    "type": "string"
                },
                "raft_id": {
                    "type": "integer"
                },
                "role": {
                    "$ref": "#/definitions/domain.ClusterRole"
                },
                "type": {
                    "$ref": "#/definitions/domain.ClusterType"
                },
                "updated_at": {
                    "type": "string"
                }
            }
        },
        "domain.ClusterRole": {
            "type": "string",
            "enum": [
                "master",
                "worker"
            ],
            "x-enum-varnames": [
                "ClusterRoleMaster",
                "ClusterRoleWorker"
            ]
        },
        "domain.ClusterType": {
            "type": "string",
            "enum": [
                "cloud",
                "fog",
                "edge"
            ],
            "x-enum-varnames": [
                "ClusterTypeCloud",
                "ClusterTypeFog",
                "ClusterTypeEdge"
            ]
        },
        "domain.Labels": {
            "type": "object",
            "additionalProperties": {
                "type": "string"
            }
        },
        "domain.Node": {
            "type": "object",
            "properties": {
                "cluster_id": {
                    "type": "string"
                },
                "cpu": {
                    "type": "integer"
                },
                "cpu_arch": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "energy": {
                    "type": "number"
                },
                "ephemeral_storage": {
                    "type": "number"
                },
                "is_disk_pressure_exists": {
                    "type": "boolean"
                },
                "is_memory_pressure_exists": {
                    "type": "boolean"
                },
                "is_pid_pressure_exists": {
                    "type": "boolean"
                },
                "is_ready": {
                    "type": "boolean"
                },
                "is_schedulable": {
                    "type": "boolean"
                },
                "location": {
                    "type": "string"
                },
                "memory": {
                    "type": "number"
                },
                "name": {
                    "type": "string"
                },
                "network_bandwidth": {
                    "type": "number"
                },
                "pricing": {
                    "type": "number"
                },
                "updated_at": {
                    "type": "string"
                }
            }
        },
        "domain.QoS": {
            "type": "object",
            "properties": {
                "energy": {
                    "type": "number"
                },
                "performance": {
                    "type": "number"
                },
                "pricing": {
                    "type": "number"
                }
            }
        },
        "dto.AddClusterReq": {
            "type": "object",
            "properties": {
                "ip": {
                    "type": "string"
                },
                "raft_id": {
                    "type": "integer"
                },
                "role": {
                    "$ref": "#/definitions/domain.ClusterRole"
                },
                "type": {
                    "$ref": "#/definitions/domain.ClusterType"
                }
            }
        },
        "dto.Application": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "string"
                },
                "labels": {
                    "$ref": "#/definitions/domain.Labels"
                },
                "name": {
                    "type": "string"
                },
                "svc": {
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                },
                "svc_nodes": {
                    "type": "string"
                },
                "svc_version": {
                    "type": "string"
                }
            }
        },
        "dto.ClusterApplicationStatus": {
            "type": "object",
            "properties": {
                "application_version": {
                    "type": "string"
                },
                "message": {
                    "type": "string"
                },
                "status": {
                    "$ref": "#/definitions/domain.ApplicationState"
                }
            }
        },
        "dto.ScheduledApplication": {
            "type": "object",
            "properties": {
                "application": {
                    "$ref": "#/definitions/dto.Application"
                },
                "cloud_ip": {
                    "type": "string"
                },
                "edge_ip": {
                    "type": "string"
                },
                "fog_ip": {
                    "type": "string"
                }
            }
        },
        "dto.UpdateClusterReq": {
            "type": "object",
            "properties": {
                "ip": {
                    "type": "string"
                },
                "raft_id": {
                    "type": "integer"
                },
                "role": {
                    "$ref": "#/definitions/domain.ClusterRole"
                },
                "type": {
                    "$ref": "#/definitions/domain.ClusterType"
                }
            }
        },
        "dto.UpdateClusterResp": {
            "type": "object",
            "properties": {
                "cluster_config": {
                    "type": "object",
                    "additionalProperties": {
                        "type": "string"
                    }
                },
                "created_at": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "ip": {
                    "type": "string"
                },
                "type": {
                    "$ref": "#/definitions/domain.ClusterType"
                },
                "updated_at": {
                    "type": "string"
                }
            }
        },
        "dto.UpsertNode": {
            "type": "object",
            "properties": {
                "nodes": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.Node"
                    }
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
