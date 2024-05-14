// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

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
        "/chats/create-chat": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Chats"
                ],
                "summary": "Создание чата",
                "parameters": [
                    {
                        "description": "Данные для создания чата",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/chats.CreateChatStruct"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "successful response",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "400": {
                        "description": "bad request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/chats/find-chats": {
            "get": {
                "description": "Получает список пользователей",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Chats"
                ],
                "summary": "Поиск пользователей",
                "parameters": [
                    {
                        "type": "string",
                        "description": "search_term",
                        "name": "search_term",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "UUID пользователя",
                        "name": "uuid",
                        "in": "query",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "successful response",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "400": {
                        "description": "bad request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/chats/get-chats": {
            "get": {
                "description": "Получает список чатов для указанного пользователя.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Chats"
                ],
                "summary": "Получение чатов пользователя",
                "parameters": [
                    {
                        "type": "string",
                        "description": "user_id",
                        "name": "user_id",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "UUID пользователя",
                        "name": "uuid",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Стейт следющей страницы",
                        "name": "page_state",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "successful response",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "400": {
                        "description": "bad request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/chats/get-chats-secured": {
            "get": {
                "description": "Получает список чатов для указанного пользователя.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Chats"
                ],
                "summary": "Получение чатов пользователя",
                "parameters": [
                    {
                        "type": "string",
                        "description": "user_id",
                        "name": "user_id",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "UUID пользователя",
                        "name": "uuid",
                        "in": "query",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "successful response",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "400": {
                        "description": "bad request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/token/generateToken": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Tokens"
                ],
                "summary": "Получение токена и uuid",
                "responses": {
                    "200": {
                        "description": "successful response",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/user/check-uniqueness-registration-data": {
            "post": {
                "description": "Проверяет не заняты ли login и Nik",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Users"
                ],
                "summary": "Проверка login и Nik",
                "parameters": [
                    {
                        "description": "Данные о пользователе",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/users.UserCUD"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "User registered successfully",
                        "schema": {
                            "$ref": "#/definitions/users.ResponseMessage"
                        }
                    }
                }
            }
        },
        "/user/chek-token": {
            "post": {
                "description": "Проверяет валидность пользователя =\u003e сессии",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Users"
                ],
                "summary": "Проверка токена",
                "parameters": [
                    {
                        "description": "Данные о токене",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/users.UserCT"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "User registered successfully",
                        "schema": {
                            "$ref": "#/definitions/users.ResponseMessage"
                        }
                    },
                    "400": {
                        "description": "Invalid request data",
                        "schema": {
                            "$ref": "#/definitions/users.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Failed to connect to database",
                        "schema": {
                            "$ref": "#/definitions/users.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/user/log-in-with-credentials": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Users"
                ],
                "summary": "Аутентификация пользователя по логину и паролю",
                "parameters": [
                    {
                        "description": "Данные пользователя",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/users.UserLogin"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "successful response",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "400": {
                        "description": "bad request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/user/registration": {
            "post": {
                "description": "Регистрирует пользователя, сохраняя зашифрованные данные и ключ в базу данных.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Users"
                ],
                "summary": "Регистрация нового пользователя",
                "parameters": [
                    {
                        "description": "Данные пользователя",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/users.UserRegistration"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "User registered successfully",
                        "schema": {
                            "$ref": "#/definitions/users.ResponseMessage"
                        }
                    },
                    "400": {
                        "description": "Invalid request data",
                        "schema": {
                            "$ref": "#/definitions/users.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Failed to connect to database",
                        "schema": {
                            "$ref": "#/definitions/users.ErrorResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "chats.CreateChatStruct": {
            "description": "Данные для создания чатов",
            "type": "object",
            "properties": {
                "companion_id": {
                    "type": "string"
                },
                "user_id": {
                    "type": "string"
                },
                "uuid": {
                    "type": "string"
                }
            }
        },
        "users.ErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string"
                },
                "status": {
                    "type": "boolean"
                }
            }
        },
        "users.ResponseMessage": {
            "type": "object",
            "properties": {
                "Data": {
                    "type": "object",
                    "additionalProperties": true
                },
                "message": {
                    "type": "string"
                },
                "status": {
                    "type": "boolean"
                }
            }
        },
        "users.UserCT": {
            "description": "User chekToken data structure.",
            "type": "object",
            "properties": {
                "pKey": {
                    "type": "string"
                },
                "token": {
                    "type": "string"
                },
                "uuid": {
                    "type": "string"
                }
            }
        },
        "users.UserCUD": {
            "description": "Данные для проверки на уникальность данных",
            "type": "object",
            "properties": {
                "Login": {
                    "type": "string"
                },
                "Nik": {
                    "type": "string"
                },
                "uuid": {
                    "type": "string"
                }
            }
        },
        "users.UserLogin": {
            "description": "User login data structure.",
            "type": "object",
            "properties": {
                "login": {
                    "type": "string",
                    "example": "john_doe"
                },
                "pKey": {
                    "type": "string"
                },
                "password": {
                    "type": "string",
                    "example": "securePassword123"
                },
                "uuid": {
                    "type": "string"
                }
            }
        },
        "users.UserRegistration": {
            "description": "User registration data structure.",
            "type": "object",
            "properties": {
                "login": {
                    "type": "string",
                    "example": "john_doe"
                },
                "name": {
                    "type": "string",
                    "example": "John"
                },
                "nik": {
                    "type": "string",
                    "example": "JohnnyD"
                },
                "pKey": {
                    "type": "string"
                },
                "password": {
                    "type": "string",
                    "example": "securePassword123"
                },
                "soName": {
                    "type": "string",
                    "example": "Doe"
                },
                "uuid": {
                    "type": "string"
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
