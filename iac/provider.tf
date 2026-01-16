terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 6.0.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }

  backend "gcs" {
  bucket = "project-3335b451-2ffb-4ece-8cd-tfstate"
  prefix = "terraform/state"
  }
}

provider "google" {
  project = local.project_id
  region  = local.location
}
