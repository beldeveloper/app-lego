# App Lego

## Description

The idea of this project is to create a tool that lets building the 3rd party applications using specific VCS branches.
Such an approach provides the QA team with the possibility to test the particular feature separately from the others.

## Technical requirements

The application requires Docker and Postgres.

## Environment variables

- APP_LEGO_HTTP_PORT
- APP_LEGO_WORKING_DIR
- APP_LEGO_DB_HOST
- APP_LEGO_DB_PORT
- APP_LEGO_DB_USER
- APP_LEGO_DB_PASSWORD
- APP_LEGO_DB_NAME
- APP_LEGO_DB_SCHEMA

## Configuration variables

- REPOSITORY_ID
- REPOSITORY_TYPE
- REPOSITORY_NAME
- REPOSITORY_ALIAS
- BRANCH_ID
- BRANCH_TYPE
- BRANCH_NAME
- BRANCH_HASH
- DEPLOYMENT_ID