# Portainer MCP

## Overview

Portainer MCP is a work in progress implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) for Portainer environments. This project aims to provide a standardized way to connect Portainer's container management capabilities with AI models and other services.

MCP (Model Context Protocol) is an open protocol that standardizes how applications provide context to LLMs (Large Language Models). Similar to how USB-C provides a standardized way to connect devices to peripherals, MCP provides a standardized way to connect AI models to different data sources and tools.

This implementation focuses on exposing Portainer environment data through the MCP protocol, allowing AI assistants and other tools to interact with your containerized infrastructure in a secure and standardized way.

*Note: This project is currently under development.*

It is currently designed to work with a Portainer administrator API token.