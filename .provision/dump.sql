CREATE TABLE "public"."repositories" (
    "id" SERIAL NOT NULL,
    "type" CHARACTER VARYING(10) NOT NULL,
    "alias" CHARACTER VARYING(200) NOT NULL,
    "name" CHARACTER VARYING(200) NOT NULL,
    "status" CHARACTER VARYING(20) NOT NULL,
    "updated_at" TIMESTAMP NOT NULL,
    PRIMARY KEY ("id")
);

CREATE TABLE "public"."branches" (
     "id" SERIAL NOT NULL,
     "repository_id" BIGINT NOT NULL,
     "type" CHARACTER VARYING(10) NOT NULL,
     "name" CHARACTER VARYING(200) NOT NULL,
     "hash" CHARACTER VARYING(200) NOT NULL,
     "status" CHARACTER VARYING(20) NOT NULL,
     PRIMARY KEY ("id")
);

CREATE TABLE "public"."deployments" (
     "id" SERIAL NOT NULL,
     "status" CHARACTER VARYING(20) NOT NULL,
     "created_at" TIMESTAMP NOT NULL,
     "auto_rebuild" BOOLEAN NOT NULL,
     "branches" JSONB,
     PRIMARY KEY ("id")
);
