# go-tdms

This is a pure Go no-dependency* file parser for the Technical Data Management Streaming (TDMS) format used by National Instruments (NI) software such as LabVIEW.

*There is a single dependency that's used only for tests, so this shouldn't be brought through into your code when you install it.

## Status

As of February 2026, this is being actively maintained but has not been battled-tested.

| Feature | Status |
| ------- | ------ |
| Reading TDMS file metadata | ☑️ |
| Reading properties from file objects | ☑️ |
| Reading data from channels | ☑️ |
| Streaming data from channels | ☑️ |
| Extended precision floating point data type | ☑️ |
| Timestamp floating point data type | ☑️ |
| Data scaling | □ |
| DAQmx data and data scaling | □ |

## References

I used a few bits of code and documentation to write this, such as:

- https://www.ni.com/en/support/documentation/supplemental/06/the-ni-tdms-file-format.html
- https://www.ni.com/en/support/documentation/supplemental/07/tdms-file-format-internal-structure.html
- https://www.ni.com/docs/en-US/bundle/labview/page/tdm-data-model.html
- https://www.ni.com/en/support/documentation/supplemental/06/introduction-to-labview-tdm-streaming-vis.html
- https://www.ni.com/docs/en-US/bundle/labwindows-cvi/page/cvi/libref/cvitdmslibrary.htm
- https://github.com/ni/nidaqmx-python
- https://github.com/adamreeve/npTDMS/
