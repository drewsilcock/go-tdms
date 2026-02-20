# Changelog

## v0.3.0 – 20th February 2026

✨ Features:

- Add `Waveform` for generating x-axis values for a given channel.
- Add `TimeWaveform` for when you know that your channels use time domain.
- Add `Channel.Waveform()` method for retrieving waveform from channel.
- Add `Properties.GetTimestamp()` to get property as timestamp.
- Add `Channel.Unit()` to easily get the unit for a given channel from the standard `unit_string` property used by LabVIEW. Returns empty string if unit string property not found or not a string.

💥 Breaking changes:

- No

🐛 Bug fixes:

- Also no

🦋 New bugs introduced:

- Hopefully not

## v0.2.0 – 19th February 2026

✨ Features:

- Implement support for DAQmx and data scaling.
- Lots of improvement to API and internals.
- Lots more tests.

💥 Breaking changes:

- Yes

🐛 Bug fixes:

- Also yes

## v0.1.0 – 6th February 2026

Initial version of the package, with support for full and index TDMS files and all data types apart from fixed point and DAQmx.

Supports streaming data for a channel from file and reading all data into a single slice.
