package tdms

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
)

// NI have a bunch of scaling functions, read from the property
// "NI_Scale[i]_Scale_Type" where i is the scale index. This only applies if
// "NI_Scaling_Status" is "unscaled", confusingly. If "NI_Scaling_Status" if
// "scaled", it means the data has been saved pre-scaled and so does not need
// scaling again.
//
// DAQmxScaling works differently.
//
// We support the following scaling functions:
//
//   - polynomial
//   - linear
//   - RTD (Resistance Temperature Detector)
//   - strain
//   - table
//   - thermistor
//   - thermocouple
//   - add (not mentioned in NI docs)
//   - subtract (not mentioned in NI docs)
//   - advanced API (taken as no-op, not mentioned in NI docs)
//   - reciprocal
//
// See:
//
//   - https://www.ni.com/docs/en-US/bundle/labview-api-ref/page/functions/tdms-set-properties.html
//   - https://www.ni.com/docs/en-US/bundle/labwindows-cvi/page/cvi/libref/cvitdmslibraryfunctiontree.htm
//   - https://www.ni.com/docs/en-US/bundle/labview-api-ref/page/vi-lib/utility/tdmsutil-llb/tdms-create-scaling-information-vi.html
//
// TODO:
//
// Currently, `Scale()` returns a new output array. It might be better to re-use
// the input array to avoid allocating another big slice?

type Numeric interface {
	~int8 | ~int16 | ~int32 | ~int64 |
		~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

type excitationType int
type resistanceConfiguration int
type thermocoupleType int

const (
	scaleIndexRawDataInput int = 0xff_ff_ff_ff

	excitationTypeVoltage = 10322
	excitationTypeCurrent = 10134

	// 2-wire mode (default)
	resistanceConfiguration2Wire = 2

	// 3-wire mode
	resistanceConfiguration3Wire = 3

	// 4-wire mode
	resistanceConfiguration4Wire = 4

	thermocoupleTypeB thermocoupleType = 10047
	thermocoupleTypeE thermocoupleType = 10055
	thermocoupleTypeJ thermocoupleType = 10072
	thermocoupleTypeK thermocoupleType = 10073
	thermocoupleTypeN thermocoupleType = 10077
	thermocoupleTypeR thermocoupleType = 10082
	thermocoupleTypeS thermocoupleType = 10085
	thermocoupleTypeT thermocoupleType = 10086
)

type ScaleType string

const (
	ScaleTypePolynomial   ScaleType = "Polynomial"
	ScaleTypeLinear       ScaleType = "Linear"
	ScaleTypeRTD          ScaleType = "RTD"
	ScaleTypeStrain       ScaleType = "Strain"
	ScaleTypeTable        ScaleType = "Table"
	ScaleTypeThermistor   ScaleType = "Thermistor"
	ScaleTypeThermocouple ScaleType = "Thermocouple"
	ScaleTypeAdd          ScaleType = "Add"
	ScaleTypeSubtract     ScaleType = "Subtract"
	ScaleTypeAdvancedAPI  ScaleType = "AdvancedAPI"
	ScaleTypeReciprocal   ScaleType = "Reciprocal"

	// These are DAQmx scalers so this string never appears in any TDMS files
	// but we include them here to consistency and to adhere to the [Scaler]
	// interface.
	ScaleTypeDAQmxFormatChanging ScaleType = "DAQmxFormatChanging"
	ScaleTypeDAQmxDigitalLine    ScaleType = "DAQmxDigitalLine"
)

var scaleTypeRegex = regexp.MustCompile(`^NI_Scale\[(\d+)\]_Scale_Type$`)

type Scaler interface {
	ReadProperties(props Properties, scaleIndex int) error

	// Scale applies scaling as defined by the individual scalers and its param
	// in-line to the input values.
	//
	// Scale is likely to change the values of the outgoing values depending on
	// the specific scaler. Add and Subtract calculate the type from the two
	// input types, no-op scaling maintains input type, DAQmx scalers maintain
	// input type, and all others produce float64 values.
	//
	// Scale reserves the right to return a reference to the original input
	// slice even this is more efficient, otherwise it is likely to allocate a
	// new slice for the return data.
	//
	// Invoking this function with anything other than a slice as input is
	// invalid and will result in a panic.
	Scale(any, ...any) (any, error)

	// Type returns the type of the scaler.
	Type() ScaleType
}

type baseScaler struct {
	// inputSource is the scale index that should be used to feed this scaler.
	// When there are multiple scalers for the same data, they are recursively
	// computed starting form the final scaler and going back till you reach the
	// scaler that takes raw data as input, then passing the data back up the
	// chain.
	inputSource int
}

func (s *baseScaler) ReadProperties(props Properties, scaleIndex int, scaleName string) error {
	var err error

	// If source is not specified, fall back to raw data from channel as
	// being the source (as is common for scale at index 0).
	s.inputSource, err = props.GetInt(
		fmt.Sprintf("NI_Scale[%d]_%s_Input_Source", scaleIndex, scaleName),
		scaleIndexRawDataInput,
	)
	if err != nil {
		return fmt.Errorf("failed to read input source for scale %d: %w", scaleIndex, err)
	}

	return nil
}

type NoOpScaler struct {
	baseScaler
	name string
}

func (s *NoOpScaler) ReadProperties(props Properties, scaleIndex int) error {
	return s.baseScaler.ReadProperties(props, scaleIndex, s.name)
}

func (s *NoOpScaler) Scale(input any, _otherInputs ...any) (any, error) {
	return input, nil
}

func (s *NoOpScaler) Type() ScaleType {
	return ScaleTypeAdvancedAPI
}

type LinearScaler struct {
	baseScaler

	intercept float64
	slope     float64
}

func (s *LinearScaler) ReadProperties(props Properties, scaleIndex int) error {
	if err := s.baseScaler.ReadProperties(props, scaleIndex, "Linear"); err != nil {
		return err
	}

	var err error
	pref := fmt.Sprintf("NI_Scale[%d]_Linear_")

	s.intercept, err = props.GetFloat(pref + "Y_Intercept")
	if err != nil {
		return fmt.Errorf("failed to read intercept property for scale %d: %w", scaleIndex, err)
	}

	s.slope, err = props.GetFloat(pref + "Slope")
	if err != nil {
		return fmt.Errorf("failed to convert slope property to float for scale %d: %w", scaleIndex, err)
	}

	return nil
}

func (s *LinearScaler) Scale(values any, _otherInputs ...any) (any, error) {
	out := make([]float64, len(values.([]any)))

	switch v := values.(type) {
	case []int8:
		linearScale(s, v, out)
	case []int16:
		linearScale(s, v, out)
	case []int32:
		linearScale(s, v, out)
	case []int64:
		linearScale(s, v, out)
	case []uint8:
		linearScale(s, v, out)
	case []uint16:
		linearScale(s, v, out)
	case []uint32:
		linearScale(s, v, out)
	case []uint64:
		linearScale(s, v, out)
	case []float32:
		linearScale(s, v, out)
	case []float64:
		linearScale(s, v, out)
	default:
		return nil, fmt.Errorf("unsupported type for linear scaling: %T", v)
	}
	return nil, nil
}

func (s *LinearScaler) Type() ScaleType {
	return ScaleTypeLinear
}

func linearScale[T Numeric](s *LinearScaler, values []T, out []float64) {
	for i, v := range values {
		out[i] = float64(v)*s.slope + s.intercept
	}
}

type PolynomialScaler struct {
	baseScaler
	coefficients []float64
}

func (s *PolynomialScaler) ReadProperties(props Properties, scaleIndex int) error {
	if err := s.baseScaler.ReadProperties(props, scaleIndex, "Polynomial"); err != nil {
		return err
	}

	pref := fmt.Sprintf("NI_Scale[%d]_Polynomial_")

	numCoefficients, err := props.GetUint(pref + "Coefficients_Size")
	if errors.Is(err, ErrPropertyNotFound) {
		// Fall back to 4, following npTDMS behaviour. Not sure whether this is
		// based on anything legit or not.
		numCoefficients = 4
	} else if err != nil {
		return fmt.Errorf("failed to read number of coefficients: %w", err)
	}

	s.coefficients = make([]float64, numCoefficients)
	for i := range numCoefficients {
		s.coefficients[i], err = props.GetFloat(fmt.Sprintf("%sCoefficients[%d]", pref, i))
		if err != nil {
			return fmt.Errorf("failed to read coefficient %d: %w", i, err)
		}
	}

	return nil
}

func (s *PolynomialScaler) Scale(values any, _otherInputs ...any) (any, error) {
	out := make([]float64, len(values.([]any)))

	switch v := values.(type) {
	case []int8:
		polynomialScale(s, v, out)
	case []int16:
		polynomialScale(s, v, out)
	case []int32:
		polynomialScale(s, v, out)
	case []int64:
		polynomialScale(s, v, out)
	case []uint8:
		polynomialScale(s, v, out)
	case []uint16:
		polynomialScale(s, v, out)
	case []uint32:
		polynomialScale(s, v, out)
	case []uint64:
		polynomialScale(s, v, out)
	case []float32:
		polynomialScale(s, v, out)
	case []float64:
		polynomialScale(s, v, out)
	default:
		return nil, fmt.Errorf("invalid input type: %T", v)
	}

	return out, nil
}

func (s *PolynomialScaler) Type() ScaleType {
	return ScaleTypePolynomial
}

// Calculate c[0] + c[1]*x + c[2]*x^2 + ... + c[N-1]*x^(N-1) where c is the
// slice of coefficients and N is the number of coefficients. This
// replicates behaviour of numpy.polynomial.polynomial.polyval.
func polynomialScale[T Numeric](s *PolynomialScaler, values []T, out []float64) {
	for coeffIdx, coeff := range s.coefficients {
		for outIdx := range out {
			out[outIdx] += coeff * math.Pow(float64(values[outIdx]), float64(coeffIdx))
		}
	}
}

// RTDScaler is a scaler for resistance temperature detectors (RTDs).
//
// This scaler uses the Callendar-Van Dusen equation to convert the input signal
// from volts to temperature (measured in degrees Celsius).
type RTDScaler struct {
	baseScaler
	currentExcitation   float64
	r0NominalResistance float64
	a                   float64
	b                   float64
	c                   float64
	leadWireResistance  float64
	resistanceConfig    resistanceConfiguration
}

func (s *RTDScaler) ReadProperties(props Properties, scaleIndex int) error {
	if err := s.baseScaler.ReadProperties(props, scaleIndex, "RTD"); err != nil {
		return err
	}

	var err error
	pref := fmt.Sprintf("NI_Scale[%d]_RTD_", scaleIndex)

	s.currentExcitation, err = props.GetFloat(pref + "Current_Excitation")
	if err != nil {
		return fmt.Errorf("failed to read current excitation: %w", err)
	}

	s.r0NominalResistance, err = props.GetFloat(pref + "R0_Nominal_Resistance")
	if err != nil {
		return fmt.Errorf("failed to read R0 nominal resistance: %w", err)
	}

	s.a, err = props.GetFloat(pref + "A")
	if err != nil {
		return fmt.Errorf("failed to read A: %w", err)
	}

	s.b, err = props.GetFloat(pref + "B")
	if err != nil {
		return fmt.Errorf("failed to read B: %w", err)
	}

	s.c, err = props.GetFloat(pref + "C")
	if err != nil {
		return fmt.Errorf("failed to read C: %w", err)
	}

	s.leadWireResistance, err = props.GetFloat(pref + "Lead_Wire_Resistance")
	if err != nil {
		return fmt.Errorf("failed to read lead wire resistance: %w", err)
	}

	resistanceConfig, err := props.GetInt(pref + "Resistance_Configuration")
	if err != nil {
		return fmt.Errorf("failed to read resistance configuration: %w", err)
	}

	s.resistanceConfig = resistanceConfiguration(resistanceConfig)

	return nil
}

func (s *RTDScaler) Scale(values any, _otherInputs ...any) (any, error) {
	out := make([]float64, len(values.([]any)))

	switch v := values.(type) {
	case []int8:
		scaleRTD(s, v, out)
	case []int16:
		scaleRTD(s, v, out)
	case []int32:
		scaleRTD(s, v, out)
	case []int64:
		scaleRTD(s, v, out)
	case []uint8:
		scaleRTD(s, v, out)
	case []uint16:
		scaleRTD(s, v, out)
	case []uint32:
		scaleRTD(s, v, out)
	case []uint64:
		scaleRTD(s, v, out)
	case []float32:
		scaleRTD(s, v, out)
	case []float64:
		scaleRTD(s, v, out)
	default:
		return nil, fmt.Errorf("unsupported type: %T", values)
	}

	return out, nil
}

func (s *RTDScaler) Type() ScaleType {
	return ScaleTypeRTD
}

func scaleRTD[T Numeric](s *RTDScaler, values []T, out []float64) {
	// Callendar-Van Dusen equation:
	//
	// R(T) = { R(0)[1 + A*T + B*T^2]                  if T >= 0°C
	//        { R(0)[1 + A*T + B*T^2 + C*(T - 100)T^3] if T < 0°C

	for i := range out {
		// If R(T) >= R(0), that means T is more than zero and therefore we can
		// use the simpler quadratic form which is solved using quadratic
		// formula:
		//
		// BT^2 + A^T + 1 - R(T)/R(0) = 0
		// T = [-A ± sqrt(A^2 - 4*B*(1 - R(T)/R(0)))] / (2*B)
		//
		// Remember:
		// R = V / I, input is volts, current is from scaler config.
		rt := float64(values[i]) / s.currentExcitation
		r0 := s.r0NominalResistance

		// I'm not entirely sure what this is doing physically.
		rt = adjustForLeadResistance(
			rt,
			excitationTypeCurrent,
			s.resistanceConfig,
			s.leadWireResistance,
		)

		out[i] = (-s.a + math.Sqrt(s.a*s.a-4*s.b*(1-(rt/r0)))) / (2 * s.b)
		if rt >= r0 {
			continue
		}

		// This means we need to use the full quartic version – we have a good
		// initial guess using the quadratic form, so we can just do a few
		// iterations of Newton-Raphson to improve on it.
		temp := out[i]
		for range 5 {
			// f(T) = R(0)[1 + A*T + B*T^2 + C*(T - 100)T^3] - R(T)
			// df(T)/dT = R(0)[A + 2*B*T + C*(4*T^3 - 300*T^2)]
			f := r0*(1+s.a*temp+s.b*temp*temp+s.c*(temp-100)*temp*temp*temp) - rt
			df := r0 * (s.a + 2*s.b*temp + s.c*(4*temp-300)*temp*temp)
			temp = temp - f/df
		}

		out[i] = temp
	}
}

type StrainScalerConfig uint64

const (
	StrainScalerConfigFullBridge1    StrainScalerConfig = 10183
	StrainScalerConfigFullBridge2    StrainScalerConfig = 10184
	StrainScalerConfigFullBridge3    StrainScalerConfig = 10185
	StrainScalerConfigHalfBridge1    StrainScalerConfig = 10188
	StrainScalerConfigHalfBridge2    StrainScalerConfig = 10189
	StrainScalerConfigQuarterBridge1 StrainScalerConfig = 10271
	StrainScalerConfigQuarterBridge2 StrainScalerConfig = 10272
)

// StrainScaler converts input voltage data into strain.
//
// See:
// https://www.ni.com/docs/en-US/bundle/labview-api-ref/page/vi-lib/utility/tdmsutil-llb/tdms-create-scaling-information-strain-vi.html
type StrainScaler struct {
	baseScaler
	configuration        StrainScalerConfig
	poissonRatio         float64
	gageResistance       float64
	leadWireResistance   float64
	initialBridgeVoltage float64
	gageFactor           float64
	gainAdjustment       float64
	voltageExcitation    float64
}

func (s *StrainScaler) ReadProperties(props Properties, scaleIndex int) error {
	if err := s.baseScaler.ReadProperties(props, scaleIndex, "Strain"); err != nil {
		return err
	}

	pref := fmt.Sprintf("NI_Scale[%d]_Strain_", scaleIndex)

	config, err := props.GetUint(pref + "Configuration")
	if err != nil {
		return fmt.Errorf("failed to read configuration: %w", err)
	}
	s.configuration = StrainScalerConfig(config)

	s.poissonRatio, err = props.GetFloat(pref + "Poisson_Ratio")
	if err != nil {
		return fmt.Errorf("failed to read Poisson ratio: %w", err)
	}

	s.gageResistance, err = props.GetFloat(pref + "Gage_Resistance")
	if err != nil {
		return fmt.Errorf("failed to read gage resistance: %w", err)
	}

	s.leadWireResistance, err = props.GetFloat(pref + "Lead_Wire_Resistance")
	if err != nil {
		return fmt.Errorf("failed to read lead wire resistance: %w", err)
	}

	s.initialBridgeVoltage, err = props.GetFloat(pref + "Initial_Bridge_Voltage")
	if err != nil {
		return fmt.Errorf("failed to read initial bridge voltage: %w", err)
	}

	s.gageFactor, err = props.GetFloat(pref + "Gage_Factor")
	if err != nil {
		return fmt.Errorf("failed to read gage factor: %w", err)
	}

	s.gainAdjustment, err = props.GetFloat(pref + "Bridge_Shunt_Calibration_Gain_Adjustment")
	if err != nil {
		return fmt.Errorf("failed to read gain adjustment: %w", err)
	}

	s.voltageExcitation, err = props.GetFloat(pref + "Voltage_Excitation")
	if err != nil {
		return fmt.Errorf("failed to read voltage excitation: %w", err)
	}

	return nil
}

func (s *StrainScaler) Scale(input any, _otherInputs ...any) (any, error) {
	out := make([]float64, len(input.([]any)))
	var err error

	switch v := input.(type) {
	case []int8:
		err = strainScale(s, v, out)
	case []int16:
		err = strainScale(s, v, out)
	case []int32:
		err = strainScale(s, v, out)
	case []int64:
		err = strainScale(s, v, out)
	case []uint8:
		err = strainScale(s, v, out)
	case []uint16:
		err = strainScale(s, v, out)
	case []uint32:
		err = strainScale(s, v, out)
	case []uint64:
		err = strainScale(s, v, out)
	case []float32:
		err = strainScale(s, v, out)
	case []float64:
		err = strainScale(s, v, out)
	default:
		return nil, fmt.Errorf("unsupported input type: %T", v)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to scale input: %w", err)
	}

	return out, nil
}

func (s *StrainScaler) Type() ScaleType {
	return ScaleTypeStrain
}

func strainScale[T Numeric](s *StrainScaler, values []T, out []float64) error {
	// I'm not going to pretend to understand the reasoning behind these
	// calculations – we use npTDMS as the guiding voice to help us through
	// these dark, undocumented features. Take my documented notes below with a
	// (large) pinch of salt given I'm not an expert in this field.

	// Input data values to this function are output voltages from the
	// Wheatstone bridge, i.e. Vₒ where Vₒ is:
	// Vₒ = [R3 / (R3 + R4) - R2 / (R1 + R2)] Vex
	// Where Vex is the excitation voltage and R1-R4 are the resistances from
	// each resistor in the bridge.

	for i, v := range values {
		vo := float64(v) - s.initialBridgeVoltage
		g := s.gageFactor
		vex := s.voltageExcitation
		nu := s.poissonRatio

		// Gain adjustment is applied after all the other calculations as multiplication.
		ga := s.gainAdjustment

		// For non-full bridge setups, we apply lead wire resistance adjustment factor, which is:
		// la = 1 + Rlw / Rg
		// Where Rlw is the lead wire resistance, Rg is the gage resistance, and
		// la is the adjustment factor. This is done at the same time as the
		// gain adjustment, after the other calculations, again as a
		// multiplication.
		la := 1 + s.leadWireResistance/s.gageResistance

		// All the different configurations are explained here:
		// https://www.ni.com/en/shop/data-acquisition/sensor-fundamentals/measuring-strain-with-strain-gages.html
		switch s.configuration {
		case StrainScalerConfigFullBridge1:
			// Full bridge type I configuration:
			// R1 = R3 = R0 (1 - ε·G)
			// R2 = R4 = R0 (1 + ε·G)
			// Where R0 = unstrained resistance, G = gauge factor, and ε is the strain.
			// Substituting into bridge equation gives:
			// Vₒ = -ε·G·Vex
			// ε = - Vₒ / G·Vex
			out[i] = -vo / (g * vex)
		case StrainScalerConfigFullBridge2:
			// Full bridge type II configuration:
			// R1 = R0 (1 - ε·ν·G)
			// R2 = R0 (1 + ε·ν·G)
			// R3 = R0 (1 - ε·G)
			// R4 = R0 (1 + ε·G)
			// Where ν is the Poisson ratio.
			// Substituting gives:
			// Vₒ = - (1/2) ε·G·Vex (1 + ν)
			// ε = -2·Vₒ / (G·Vex (1 + ν))
			out[i] = -2 * vo / (g * vex * (1 + nu))
		case StrainScalerConfigFullBridge3:
			// Full bridge type III configuration:
			// R1 = R3 = R0 (1 - ε·ν·G)
			// R2 = R4 = R0 (1 + ε·G)
			// Substituting gives:
			// Vₒ = -ε·G·Vex (1 + ν) Vex / (2 + ε·G (1 - v))
			// ε = -2·Vₒ / [G (Vₒ (1 - ν) + Vex (1 + ν))]
			out[i] = -2 * vo / (g * (vo*(1-nu) + vex*(1+nu)))
		case StrainScalerConfigHalfBridge1:
			// Half bridge type I configuration:
			// R1 = R2 = R0
			// R3 = R0 (1 - ε·ν·G)
			// R4 = R0 (1 + ε·G)
			// Substituting gives:
			// Vₒ = [(1 - ε·ν·G) / (2 + ε·G - ε·ν·G - 1/2)] Vex
			// Rearranging gives:
			// ε = -4 (Vₒ / Vex) / [G (1 + ν + 2 (Vₒ / Vex) (1 - ν))]
			out[i] = -4 * (vo / vex) / (g * (1 + nu + 2*(vo/vex)*(1-nu)))
			out[i] *= la
		case StrainScalerConfigHalfBridge2:
			// R1 = R2 = R0
			// R3 = R0 (1 - ε·G)
			// R4 = R0 (1 + ε·G)
			out[i] = -2 * vo / (g * vex)
			out[i] *= la
		case StrainScalerConfigQuarterBridge1, StrainScalerConfigQuarterBridge2:
			// Quarter bridge II is the same as quarter bridge I but with an
			// additional strain gage used to reduce the effects of temperature
			// on the strain measurements – the equations for both are
			// identical.
			// R1 = R2 = R3 = R0
			// R4 = R0 (1 + ε·G)
			// Substituting gives:
			// Vₒ = [1 / (2 + ε·G) - 1/2] Vex
			// Rearranging gives:
			// ε = (2 / G) · [1 / (1 + 2 Vₒ / Vex) - 1]
			out[i] = (2 / g) * (1/(1+2*vo/vex) - 1)
			out[i] *= la
		default:
			return fmt.Errorf("unsupported strain gauge configuration: %d", s.configuration)
		}

		out[i] *= ga
	}

	return nil
}

type TableScaler struct {
	baseScaler
	inputValues  []float64
	outputValues []float64
}

func (s *TableScaler) ReadProperties(props Properties, scaleIndex int) error {
	if err := s.baseScaler.ReadProperties(props, scaleIndex, "Table"); err != nil {
		return err
	}

	pref := fmt.Sprintf("NI_Scale[%d]_Table_", scaleIndex)

	numPreScaledValues, err := props.GetUint(pref + "Pre_Scaled_Values_Size")
	if err != nil {
		return fmt.Errorf("failed to read number of pre-scaled values: %w", err)
	}

	s.inputValues = make([]float64, numPreScaledValues)

	for i := range numPreScaledValues {
		s.inputValues[i], err = props.GetFloat(fmt.Sprintf("%sPre_Scaled_Values[%d]", pref, i))
		if err != nil {
			return fmt.Errorf("failed to read pre-scaled value %d: %w", i, err)
		}
	}

	numScaledValues, err := props.GetUint(pref + "Scaled_Values_Size")
	if err != nil {
		return fmt.Errorf("failed to read number of scaled values: %w", err)
	}

	s.outputValues = make([]float64, numScaledValues)

	for i := range numScaledValues {
		s.outputValues[i], err = props.GetFloat(fmt.Sprintf("%sScaled_Values[%d]", pref, i))
		if err != nil {
			return fmt.Errorf("failed to read scaled value %d: %w", i, err)
		}
	}

	if len(s.inputValues) != len(s.outputValues) {
		return fmt.Errorf("input and output values must have the same length")
	}

	if len(s.inputValues) == 0 {
		return fmt.Errorf("no input values provided")
	}

	// The input values are expected to be monotonically increasing, i.e.
	// pre-sorted. If they're not, try reversing the slice in case it's been set
	// in reverse order.
	if !isMonotonicInc(s.inputValues) {
		slices.Reverse(s.inputValues)
		slices.Reverse(s.outputValues)
	}

	// If they're still not monotonically increasing, we've got a problem.
	if !isMonotonicInc(s.inputValues) {
		return fmt.Errorf("input and output values must be monotonically increasing")
	}

	return nil
}

func (s *TableScaler) Scale(input any, _otherInputs ...any) (any, error) {
	out := make([]float64, len(input.([]any)))

	switch v := input.(type) {
	case []int8:
		tableScale(s, v, out)
	case []int16:
		tableScale(s, v, out)
	case []int32:
		tableScale(s, v, out)
	case []int64:
		tableScale(s, v, out)
	case []uint8:
		tableScale(s, v, out)
	case []uint16:
		tableScale(s, v, out)
	case []uint32:
		tableScale(s, v, out)
	case []uint64:
		tableScale(s, v, out)
	case []float32:
		tableScale(s, v, out)
	case []float64:
		tableScale(s, v, out)
	default:
		return nil, fmt.Errorf("unsupported input type: %T", input)
	}

	return out, nil
}

func (s *TableScaler) Type() ScaleType {
	return ScaleTypeTable
}

func tableScale[T Numeric](s *TableScaler, input []T, out []float64) {
	interp(input, s.inputValues, s.outputValues, nil, nil)
}

// ThermistorScaler calculates the temperature of a thermistor from the resistance.
//
// Uses the Steinhart-Hart equation, factoring into account factors like lead
// wire resistance and temperature offset.
//
// See:
// https://www.ni.com/docs/en-US/bundle/labview-api-ref/page/vi-lib/utility/tdmsutil-llb/tdms-create-scaling-information-thermistor-vi.html
type ThermistorScaler struct {
	baseScaler
	excitationType          excitationType
	excitationValue         float64
	resistanceConfiguration resistanceConfiguration
	r1ReferenceResistance   float64
	leadWireResistance      float64
	a                       float64
	b                       float64
	c                       float64
	temperatureOffset       float64 // Not mentioned in the NI docs but npTDMS is probably more accurate.
}

func (s *ThermistorScaler) ReadProperties(props Properties, scaleIndex int) error {
	if err := s.baseScaler.ReadProperties(props, scaleIndex, "Thermistor"); err != nil {
		return err
	}

	pref := fmt.Sprintf("NI_Scale[%d]_Thermistor_", scaleIndex)

	excitationTypeVal, err := props.GetInt(pref + "Excitation_Type")
	if err != nil {
		return fmt.Errorf("failed to read excitation type: %w", err)
	}
	s.excitationType = excitationType(excitationTypeVal)

	s.excitationValue, err = props.GetFloat(pref + "Excitation_Value")
	if err != nil {
		return fmt.Errorf("failed to read excitation value: %w", err)
	}

	resistanceConfigurationVal, err := props.GetInt(pref + "Resistance_Configuration")
	if err != nil {
		return fmt.Errorf("failed to read resistance configuration: %w", err)
	}
	s.resistanceConfiguration = resistanceConfiguration(resistanceConfigurationVal)

	s.r1ReferenceResistance, err = props.GetFloat(pref + "R1_Reference_Resistance")
	if err != nil {
		return fmt.Errorf("failed to read R1 reference resistance: %w", err)
	}

	s.leadWireResistance, err = props.GetFloat(pref + "Lead_Wire_Resistance")
	if err != nil {
		return fmt.Errorf("failed to read lead wire resistance: %w", err)
	}

	s.a, err = props.GetFloat(pref + "A")
	if err != nil {
		return fmt.Errorf("failed to read A: %w", err)
	}

	s.b, err = props.GetFloat(pref + "B")
	if err != nil {
		return fmt.Errorf("failed to read B: %w", err)
	}

	s.c, err = props.GetFloat(pref + "C")
	if err != nil {
		return fmt.Errorf("failed to read C: %w", err)
	}

	s.temperatureOffset, err = props.GetFloat(pref + "Temperature_Offset")
	if err != nil {
		return fmt.Errorf("failed to read temperature offset: %w", err)
	}

	return nil
}

func (s *ThermistorScaler) Scale(input any, _otherInputs ...any) (any, error) {
	out := make([]float64, len(input.([]any)))
	var err error

	switch v := input.(type) {
	case []int8:
		err = thermistorScale(s, v, out)
	case []int16:
		err = thermistorScale(s, v, out)
	case []int32:
		err = thermistorScale(s, v, out)
	case []int64:
		err = thermistorScale(s, v, out)
	case []uint8:
		err = thermistorScale(s, v, out)
	case []uint16:
		err = thermistorScale(s, v, out)
	case []uint32:
		err = thermistorScale(s, v, out)
	case []uint64:
		err = thermistorScale(s, v, out)
	case []float32:
		err = thermistorScale(s, v, out)
	case []float64:
		err = thermistorScale(s, v, out)
	default:
		return nil, fmt.Errorf("unsupported input type: %T", input)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to scale input: %w", err)
	}

	return out, nil
}

func (s *ThermistorScaler) Type() ScaleType {
	return ScaleTypeThermistor
}

func thermistorScale[T Numeric](s *ThermistorScaler, input []T, out []float64) error {
	// Calculates the temperature of a thermistor from the resistance, using the Steinhart-Hart equation:
	// 1/T = A + B log R + C (log R)³
	// Where T = temperature, R = resistance, and A, B, C are Steinhart-Hart coefficients.

	for i, v := range input {
		var rt float64

		switch s.excitationType {
		case excitationTypeCurrent:
			rt = float64(v) / s.excitationValue
		case excitationTypeVoltage:
			// Voltage divider circuit:
			// Rt = R1 / [(Vex / Vo) - 1]
			rt = s.r1ReferenceResistance / (s.excitationValue/float64(v) - 1)
		default:
			return fmt.Errorf("unsupported excitation type: %v", s.excitationType)
		}

		rt = adjustForLeadResistance(rt, s.excitationType, s.resistanceConfiguration, s.leadWireResistance)
		logRt := math.Log(rt)

		out[i] = 1/(s.a+s.b*logRt+s.c*math.Pow(logRt, 3)) - s.temperatureOffset
	}

	return nil
}

// ThermocoupleScaler converts voltage to temperature for a thermocouple and vice-versa.
//
// If scalingDirection is 1, converts from voltage to temperature, otherwise
// converts temperature back to voltage.
//
// Input voltage is in microvolts while temperature is in degrees Celsius.
type ThermocoupleScaler struct {
	baseScaler
	thermocouple     thermocouple
	scalingDirection int
}

func (s *ThermocoupleScaler) ReadProperties(props Properties, scaleIndex int) error {
	if err := s.baseScaler.ReadProperties(props, scaleIndex, "Thermocouple"); err != nil {
		return err
	}

	pref := fmt.Sprintf("NI_Scale[%d]_Thermocouple_", scaleIndex)

	thermocoupleTypeVal, err := props.GetInt(pref+"Type", int(thermocoupleTypeJ))
	if err != nil {
		return fmt.Errorf("failed to read thermocouple type property: %w", err)
	}

	switch thermocoupleType(thermocoupleTypeVal) {
	case thermocoupleTypeB:
		s.thermocouple = thermocoupleB
	case thermocoupleTypeE:
		s.thermocouple = thermocoupleE
	case thermocoupleTypeJ:
		s.thermocouple = thermocoupleJ
	case thermocoupleTypeK:
		s.thermocouple = thermocoupleK
	case thermocoupleTypeN:
		s.thermocouple = thermocoupleN
	case thermocoupleTypeR:
		s.thermocouple = thermocoupleR
	case thermocoupleTypeS:
		s.thermocouple = thermocoupleS
	case thermocoupleTypeT:
		s.thermocouple = thermocoupleT
	}

	s.scalingDirection, err = props.GetInt(pref+"ScalingDirection", 0)
	if err != nil {
		return fmt.Errorf("failed to read scaling direction property: %w", err)
	}

	return nil
}

func (s *ThermocoupleScaler) Scale(input any, _otherInputs ...any) (any, error) {
	out := make([]float64, len(input.([]any)))

	switch v := input.(type) {
	case []int8:
		thermocoupleScale(s, v, out)
	case []int16:
		thermocoupleScale(s, v, out)
	case []int32:
		thermocoupleScale(s, v, out)
	case []int64:
		thermocoupleScale(s, v, out)
	case []uint8:
		thermocoupleScale(s, v, out)
	case []uint16:
		thermocoupleScale(s, v, out)
	case []uint32:
		thermocoupleScale(s, v, out)
	case []uint64:
		thermocoupleScale(s, v, out)
	case []float32:
		thermocoupleScale(s, v, out)
	case []float64:
		thermocoupleScale(s, v, out)
	}

	return out, nil
}

func (s *ThermocoupleScaler) Type() ScaleType {
	return ScaleTypeThermocouple
}

func thermocoupleScale[T Numeric](s *ThermocoupleScaler, input []T, out []float64) {
	// Our conversion equations use millivolts but TDMS stores as microvolts.
	for i, v := range input {
		if s.scalingDirection == 1 {
			out[i] = 1000 * s.thermocouple.temperatureToVoltage(float64(v))
		} else {
			out[i] = s.thermocouple.voltageToTemperature(float64(v) / 1000)
		}
	}
}

type AddScaler struct {
	leftInputSource  int
	rightInputSource int
}

func (s *AddScaler) ReadProperties(props Properties, scaleIndex int) error {
	pref := fmt.Sprintf("NI_Scale[%d]_Add_", scaleIndex)

	var err error

	s.leftInputSource, err = props.GetInt(pref + "Left_Operand_Input_Source")
	if err != nil {
		return fmt.Errorf("failed to read left input source: %w", err)
	}

	s.rightInputSource, err = props.GetInt(pref + "Right_Operand_Input_Source")
	if err != nil {
		return fmt.Errorf("failed to read right input source: %w", err)
	}

	return nil
}

func (s *AddScaler) Scale(input any, otherInputs ...any) (any, error) {
	if len(otherInputs) != 1 {
		return nil, errors.New("expected exactly one other input")
	}

	out := make([]any, len(input.([]any)))

	v2 := otherInputs[0]

	switch v1 := input.(type) {
	case []int8:
		addScale(v1, v2.([]int8), out)
	case []int16:
		addScale(v1, v2.([]int16), out)
	case []int32:
		addScale(v1, v2.([]int32), out)
	case []int64:
		addScale(v1, v2.([]int64), out)
	case []uint8:
		addScale(v1, v2.([]uint8), out)
	case []uint16:
		addScale(v1, v2.([]uint16), out)
	case []uint32:
		addScale(v1, v2.([]uint32), out)
	case []uint64:
		addScale(v1, v2.([]uint64), out)
	case []float32:
		addScale(v1, v2.([]float32), out)
	case []float64:
		addScale(v1, v2.([]float64), out)
	case []complex64:
		addScale(v1, v2.([]complex64), out)
	case []complex128:
		addScale(v1, v2.([]complex128), out)
	}

	return out, nil
}

func (s *AddScaler) Type() ScaleType {
	return ScaleTypeAdd
}

func addScale[T Numeric | complex64 | complex128](leftValues []T, rightValues []T, out []any) {
	for i := range leftValues {
		out[i] = leftValues[i] + rightValues[i]
	}
}

type SubtractScaler struct {
	leftInputSource  int
	rightInputSource int
}

func (s *SubtractScaler) ReadProperties(props Properties, scaleIndex int) error {
	pref := fmt.Sprintf("NI_Scale[%d]_Subtract_", scaleIndex)

	var err error

	s.leftInputSource, err = props.GetInt(pref + "Left_Operand_Input_Source")
	if err != nil {
		return fmt.Errorf("failed to read left input source: %w", err)
	}

	s.rightInputSource, err = props.GetInt(pref + "Right_Operand_Input_Source")
	if err != nil {
		return fmt.Errorf("failed to read right input source: %w", err)
	}

	return nil
}

func (s *SubtractScaler) Scale(input any, otherInputs ...any) (any, error) {
	if len(otherInputs) != 1 {
		return nil, errors.New("expected exactly one other input")
	}

	out := make([]any, len(input.([]any)))

	v2 := otherInputs[0]

	switch v1 := input.(type) {
	case []int8:
		subtractScale(v1, v2.([]int8), out)
	case []int16:
		subtractScale(v1, v2.([]int16), out)
	case []int32:
		subtractScale(v1, v2.([]int32), out)
	case []int64:
		subtractScale(v1, v2.([]int64), out)
	case []uint8:
		subtractScale(v1, v2.([]uint8), out)
	case []uint16:
		subtractScale(v1, v2.([]uint16), out)
	case []uint32:
		subtractScale(v1, v2.([]uint32), out)
	case []uint64:
		subtractScale(v1, v2.([]uint64), out)
	case []float32:
		subtractScale(v1, v2.([]float32), out)
	case []float64:
		subtractScale(v1, v2.([]float64), out)
	case []complex64:
		subtractScale(v1, v2.([]complex64), out)
	case []complex128:
		subtractScale(v1, v2.([]complex128), out)
	}

	return out, nil
}

func (s *SubtractScaler) Type() ScaleType {
	return ScaleTypeSubtract
}

func subtractScale[T Numeric | ~complex64 | ~complex128](leftValues []T, rightValues []T, out []any) {
	for i := range leftValues {
		out[i] = leftValues[i] - rightValues[i]
	}
}

// ReciprocalScaler calculates the reciprocal of the input value.
//
// See:
// https://www.ni.com/docs/en-US/bundle/labview-api-ref/page/vi-lib/utility/tdmsutil-llb/tdms-create-scaling-information-reciprocal-vi.html
type ReciprocalScaler struct{ baseScaler }

func (s *ReciprocalScaler) ReadProperties(props Properties, scaleIndex int) error {
	return s.baseScaler.ReadProperties(props, scaleIndex, "Reciprocal")
}

func (s *ReciprocalScaler) Scale(input any, _otherInputs ...any) (any, error) {
	out := make([]any, len(input.([]any)))

	switch v := input.(type) {
	case []int8:
		reciprocalScaler(v, out)
	case []int16:
		reciprocalScaler(v, out)
	case []int32:
		reciprocalScaler(v, out)
	case []int64:
		reciprocalScaler(v, out)
	case []uint8:
		reciprocalScaler(v, out)
	case []uint16:
		reciprocalScaler(v, out)
	case []uint32:
		reciprocalScaler(v, out)
	case []uint64:
		reciprocalScaler(v, out)
	case []float32:
		reciprocalScaler(v, out)
	case []float64:
		reciprocalScaler(v, out)
	case []complex64:
		reciprocalScaler(v, out)
	case []complex128:
		reciprocalScaler(v, out)
	}

	return out, nil
}

func (s *ReciprocalScaler) Type() ScaleType {
	return ScaleTypeReciprocal
}

func reciprocalScaler[T Numeric | ~complex64 | ~complex128](values []T, out []any) {
	for i, v := range values {
		if v != 0 {
			out[i] = 1 / v
		}
	}
}

type Multiscaler struct {
	scalers []Scaler
}

func (m *Multiscaler) Scale(input any) (any, error) {
	// We had to calculate scales from front to back. To do this, we start at
	// the final scale and work back recursively.
	finalScaleIndex := len(m.scalers) - 1
	return m.computeScalings(input, finalScaleIndex)
}

func (m *Multiscaler) computeScalings(input any, scaleIndex int) (any, error) {
	if scaleIndex == scaleIndexRawDataInput {
		return input, nil
	}

	if scaleIndex < 0 || scaleIndex >= len(m.scalers) {
		return nil, fmt.Errorf("invalid scale index %d", scaleIndex)
	}

	scaler := m.scalers[scaleIndex]

	// Add and subtract scalers need to be handled differently because they take
	// two input sources instead of one.
	switch s := scaler.(type) {
	case *AddScaler:
		leftInput, err := m.computeScalings(input, s.leftInputSource)
		if err != nil {
			return nil, fmt.Errorf("failed to scale left input: %w", err)
		}

		rightInput, err := m.computeScalings(input, s.rightInputSource)
		if err != nil {
			return nil, fmt.Errorf("failed to scale right input: %w", err)
		}

		return s.Scale(leftInput, rightInput)
	case *SubtractScaler:
		leftInput, err := m.computeScalings(input, s.leftInputSource)
		if err != nil {
			return nil, fmt.Errorf("failed to scale left input: %w", err)
		}

		rightInput, err := m.computeScalings(input, s.rightInputSource)
		if err != nil {
			return nil, fmt.Errorf("failed to scale right input: %w", err)
		}

		return s.Scale(leftInput, rightInput)
	default:
		return s.Scale(input)
	}
}

// getChannelScaling retrieves the scaling for a specific channel.
//
// If the scaling is defined in the channel object itself, we use that.
// Otherwise we fall back to the group object, then the file object.
//
// We assume that the scaling does not change between segments. According to the
// spec, it is possible for scalings to change between segments but in practice
// LabVIEW does not do this.
func getChannelScaling(channel *Channel, group *Group, file *File) (*Multiscaler, error) {
	channelObj := file.objects[channel.path]
	if channelScaler, err := getObjectScaling(&channelObj); err != nil {
		return nil, err
	} else if channelScaler != nil {
		return channelScaler, nil
	}

	groupObj := file.objects[group.path]
	if groupScaler, err := getObjectScaling(&groupObj); err != nil {
		return nil, err
	} else if groupScaler != nil {
		return groupScaler, nil
	}

	fileObj := file.objects[""]
	if fileScaler, err := getObjectScaling(&fileObj); err != nil {
		return nil, err
	} else if fileScaler != nil {
		return fileScaler, nil
	}

	return nil, nil
}

// This retrieves the scaling for a specific object. As TDMS supports applying
// scaling to a whole group or file which then applies to all channels inside
// that group/file, you should use [getScaling] instead to retrieve the scaling
// for actual use.
func getObjectScaling(obj *object) (*Multiscaler, error) {
	scalingType, err := obj.properties.GetString("NI_Scaling_Status", "unscaled")
	if err != nil {
		return nil, fmt.Errorf("failed to get scaling type: %w", err)
	}
	if scalingType == "scaled" {
		// If scaling type is "scaled", it confusingly means the scaling has
		// been baked into the data in the TDMS file and so we don't want to do
		// any additional scaling ourselves.
		return nil, nil
	}

	numScalers := getNumScalings(obj.properties)
	if numScalers == 0 {
		return nil, nil
	}

	scalers := make([]Scaler, numScalers)
	for scaleIndex := range scalers {
		scaleTypeProp, ok := obj.properties[fmt.Sprintf("NI_Scale[%d]_Scale_Type", scaleIndex)]
		if !ok {
			// This must be a DAQmx scaler – they get their data from object raw data index.
			if obj.index == nil || obj.index.daqmxScalerType == daqmxScalerTypeNone {
				return nil, fmt.Errorf(
					"%w: object has %d scalers but scaler %d not found",
					ErrInvalidFileFormat,
					numScalers,
					scaleIndex,
				)
			}

			scaler, ok := obj.index.daqmxScalers[scaleIndex]
			if !ok {
				return nil, fmt.Errorf(
					"%w: object has %d scalers but scaler %d not found",
					ErrInvalidFileFormat,
					numScalers,
					scaleIndex,
				)
			}

			switch obj.index.daqmxScalerType {
			case daqmxScalerTypeFormatChanging:
				scalers[scaleIndex] = ptr(formatChangingScaler(scaler))
			case daqmxScalerTypeDigitalLine:
				scalers[scaleIndex] = ptr(digitalLineScaler(scaler))
			}

			continue
		}

		scaleType, err := scaleTypeProp.AsString()
		if err != nil {
			return nil, fmt.Errorf("failed to get scale type for scaler %d", scaleIndex)
		}

		var scaler Scaler

		switch ScaleType(scaleType) {
		case ScaleTypePolynomial:
			scaler = &PolynomialScaler{}
		case ScaleTypeLinear:
			scaler = &LinearScaler{}
		case ScaleTypeRTD:
			scaler = &RTDScaler{}
		case ScaleTypeStrain:
			scaler = &StrainScaler{}
		case ScaleTypeTable:
			scaler = &TableScaler{}
		case ScaleTypeThermistor:
			scaler = &ThermistorScaler{}
		case ScaleTypeThermocouple:
			scaler = &ThermocoupleScaler{}
		case ScaleTypeAdd:
			scaler = &AddScaler{}
		case ScaleTypeSubtract:
			scaler = &SubtractScaler{}
		case ScaleTypeAdvancedAPI:
			scaler = &NoOpScaler{}
		case ScaleTypeReciprocal:
			scaler = &ReciprocalScaler{}
		default:
			return nil, fmt.Errorf("unsupported scale type: %s", scaleType)
		}

		if err := scaler.ReadProperties(obj.properties, scaleIndex); err != nil {
			return nil, fmt.Errorf("failed to read properties for scaler %d type %s: %w", scaleIndex, scaleType, err)
		}

		scalers[scaleIndex] = scaler
	}

	return &Multiscaler{scalers: scalers}, nil
}

func getNumScalings(properties Properties) int {
	numScalings, err := properties.GetInt("NI_Number_Of_Scales")
	if err == nil {
		return numScalings
	}

	// If the expected num scalings property doesn't exist or isn't valid, fall
	// back to using regex to checking the property configuration properties.
	numScalings = 0
	for key := range properties {
		if !strings.HasPrefix(key, "NI_Scale[") {
			continue
		}

		matches := scaleTypeRegex.FindStringSubmatch(key)
		if len(matches) > 1 {
			numScalings++
		}
	}

	return numScalings
}

func adjustForLeadResistance(
	measuredResistance float64,
	excitationType excitationType,
	resistanceConfiguration resistanceConfiguration,
	leadWireResistance float64,
) float64 {
	if resistanceConfiguration == resistanceConfiguration3Wire {
		return measuredResistance - leadWireResistance
	}

	if resistanceConfiguration == resistanceConfiguration2Wire && excitationType == excitationTypeCurrent {
		return measuredResistance - 2*leadWireResistance
	}

	return measuredResistance
}
