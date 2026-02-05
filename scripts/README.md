# TDMS Test File Generator

This folder contains a script for generating both test TDMS files using npTDMS and a "manifest" file which contains all the information we need to test whether the values from the generated files match the expected values.

## Is it a good idea to rely on npTDMS to be doing the right thing every time?

Practically speaking this means that if there's a bug in npTDMS, our tests will break. That's not ideal, but npTDMS is the gold standard for TDMS parsing (even the [official NI code](https://nidaqmx-python.readthedocs.io/en/stable/#tdms-logging) suggests to use it), which means I'm not too fussed about the dependency.

## What about malformed files?

We're not currently generating malformed files for testing – that could be an interesting follow-on to this testing script.

## How do I run it?

Install uv and then run `cd scripts && uv run generate-test-files.py ../tests/testdata`.
