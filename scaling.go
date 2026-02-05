package tdms

// NI have a bunch of scaling functions, read from the property
// "NI_Scale[i]_Scale_Type" where i is the scale index. This only applies if
// "NI_Scaling_Status" is "scaled", otherwise the data is not scaled.
//
// DAQmxScaling works differently.
//
// npTDMS supports the following scaling functions"
//
//   - polynomial
//   - linear
//   - RTD (convert signal from Resistance Temperature Detector (RTD) into degrees Celsius)
//   - strain
//   - table
//   - thermistor
//   - thermocouple
//   - add
//   - subtract
//   - advanced API (taken as no-op)
//
// See: https://www.ni.com/docs/en-US/bundle/labwindows-cvi/page/cvi/libref/cvitdmslibraryfunctiontree.htm
// (scroll down to "Advanced Data Scaling")
