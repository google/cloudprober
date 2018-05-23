#!/usr/bin/env sh


psql -c "create database cloudprober"

psql -c "create schema cloudprober"

psql cloudprober -c "CREATE TABLE metrics (
  time TIMESTAMP WITH TIME ZONE,
  metric_name text NOT NULL,
  value DOUBLE PRECISION ,
  labels jsonb,
  PRIMARY KEY (time, metric_name, labels)
)"