# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Displayed durations are now rounded to the nearest 15 minutes across `status`, `history`, `report` (table and PDF export), and `log` output. Stored time remains precise — rounding is display-only and does not affect logged data or interactive edit defaults.
