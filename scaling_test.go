package tdms

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var floatOpt = cmpopts.EquateApprox(1e-9, 1e-9)

func TestUnsupportedScalingType(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "This isn't a valid scaling type")

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	} else if !errors.Is(err, ErrUnsupportedScaler) {
		t.Fatalf("Expected ErrUnsupportedScaler, got %v", err)
	}

	if scaler != nil {
		t.Fatalf("Expected nil scaling, got %v", scaler)
	}
}

func TestPrescaledData(t *testing.T) {
	props := NewProperties().
		Add("NI_Scaling_Status", "scaled").
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "Linear").
		Add("NI_Scale[0]_Linear_Slope", 2.0).
		Add("NI_Scale[0]_Linear_Y_Intercept", 10.0)

	got, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(got.scalers) != 0 {
		t.Fatalf("expected no scalers, got %v", got.scalers)
	}
	if len(got.buffers) != 0 {
		t.Fatalf("expected no buffers, got %v", got.buffers)
	}
	if got.bufferSize != 0 {
		t.Fatalf("expected 0 buffer size, got %v", got.bufferSize)
	}
	if len(got.dataTypes) != 0 {
		t.Fatalf("expected no data types, got %v", got.dataTypes)
	}
	if got.outputType != DataTypeFloat64 {
		t.Fatalf("expected output data type %v, got %v", DataTypeFloat64, got.outputType)
	}
}

func TestNoOpScaler(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "AdvancedAPI").
		Add("NI_Scale[0]_AdvancedAPI_Input_Source", scaleIndexRawDataInput)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	checkScaler(t, scaler, []Scaler{&NoOpScaler{}})

	input := []float64{1, 2, 3, 4}
	want := []float64{1, 2, 3, 4}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, ok := got.([]float64); !ok {
		t.Fatalf("Expected output to be a slice of float64, got %T", got)
	}

	// No-op returns the same reference that you put in.
	if !cmp.Equal(want, got, floatOpt) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
	}
}

func TestLinearScaler(t *testing.T) {
	props := NewProperties().
		Add("NI_Scaling_Status", "unscaled").
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "Linear").
		Add("NI_Scale[0]_Linear_Slope", 2.0).
		Add("NI_Scale[0]_Linear_Y_Intercept", 10.0)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	checkScaler(t, scaler, []Scaler{&LinearScaler{}})

	input := []float64{1, 2, 3, 4}
	want := []float64{12, 14, 16, 18}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, ok := got.([]float64); !ok {
		t.Fatalf("Expected output to be a slice of float64, got %T", got)
	}

	if !cmp.Equal(want, got, floatOpt) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
	}
}

func TestPolynomialScaler(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "Polynomial").
		Add("NI_Scale[0]_Polynomial_Coefficients[0]", 10.0).
		Add("NI_Scale[0]_Polynomial_Coefficients[1]", 1.0).
		Add("NI_Scale[0]_Polynomial_Coefficients[2]", 2.0).
		Add("NI_Scale[0]_Polynomial_Coefficients[3]", 3.0)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	checkScaler(t, scaler, []Scaler{&PolynomialScaler{}})

	input := []float64{1, 2, 3, 4}
	want := []float64{16, 44, 112, 238}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, ok := got.([]float64); !ok {
		t.Fatalf("Expected output to be a slice of float64, got %T", got)
	}

	if !cmp.Equal(want, got, floatOpt) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
	}
}

func TestPolynomialScalerWithNoCoefficients(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "Polynomial").
		Add("NI_Scale[0]_Polynomial_Coefficients_Size", 0)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	checkScaler(t, scaler, []Scaler{&PolynomialScaler{}})

	input := []float64{1, 2, 3, 4}
	want := []float64{0, 0, 0, 0}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, ok := got.([]float64); !ok {
		t.Fatalf("Expected output to be a slice of float64, got %T", got)
	}

	if !cmp.Equal(want, got, floatOpt) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
	}
}

func TestPolynomialScalerWithOneCoefficient(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "Polynomial").
		Add("NI_Scale[0]_Polynomial_Coefficients_Size", 1).
		Add("NI_Scale[0]_Polynomial_Coefficients[0]", 2)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	checkScaler(t, scaler, []Scaler{&PolynomialScaler{}})

	input := []float64{1, 2, 3, 4}
	want := []float64{2, 2, 2, 2}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, ok := got.([]float64); !ok {
		t.Fatalf("Expected output to be a slice of float64, got %T", got)
	}

	if !cmp.Equal(want, got, floatOpt) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
	}
}

func TestRTDScaler(t *testing.T) {
	cases := []struct {
		name             string
		resistanceConfig int
		leadResistance   float64
		want             []float64
	}{
		{"2-wire, no lead resistance", 2, 0.0, []float64{1256.89628, 1712.83429}},
		{"2-wire, w/ lead resistance", 2, 100.0, []float64{557.6879004146, 882.7374139697}},
		{"3-wire, no lead resistance", 3, 0.0, []float64{1256.89628, 1712.83429}},
		{"3-wire, w/ lead resistance", 3, 100.0, []float64{882.7374139697, 1256.896275222}},
		{"4-wire, no lead resistance", 4, 0.0, []float64{1256.89628, 1712.83429}},
		{"4-wire, w/ lead resistance", 4, 100.0, []float64{1256.89628, 1712.83429}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := NewProperties().
				Add("NI_Number_Of_Scales", 1).
				Add("NI_Scale[0]_Scale_Type", "RTD").
				Add("NI_Scale[0]_RTD_Current_Excitation", 0.001).
				Add("NI_Scale[0]_RTD_R0_Nominal_Resistance", 100.0).
				Add("NI_Scale[0]_RTD_A", 0.0039083).
				Add("NI_Scale[0]_RTD_B", -5.775e-07).
				Add("NI_Scale[0]_RTD_C", -4.183e-12).
				Add("NI_Scale[0]_RTD_Lead_Wire_Resistance", tc.leadResistance).
				Add("NI_Scale[0]_RTD_Resistance_Configuration", tc.resistanceConfig).
				Add("NI_Scale[0]_RTD_Input_Source", scaleIndexRawDataInput)

			scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
			if err != nil {
				t.Fatalf("Error creating scaler: %v", err)
			}

			checkScaler(t, scaler, []Scaler{&RTDScaler{}})

			input := []float64{0.5, 0.6}

			if err := scaler.Allocate(len(input)); err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			got, err := scaler.Scale(input)
			if err != nil {
				t.Fatalf("Error scaling input: %v", err)
			}

			if _, ok := got.([]float64); !ok {
				t.Fatalf("Expected output to be a slice of float64, got %T", got)
			}

			// RTD calculation is particularly prone to floating point
			// imprecision, so bump up the allowed error.
			if !cmp.Equal(tc.want, got, cmpopts.EquateApprox(1e-7, 1e-7)) {
				t.Fatalf("Output differs: %v", cmp.Diff(tc.want, got, cmpopts.EquateApprox(1e-7, 1e-7)))
			}
		})
	}
}

var wantStrainOutput = map[StrainScalerConfig]map[string][]float64{
	StrainScalerConfigFullBridge1: {
		"baseline": {
			-1.310990e-03, -1.295924e-03, -1.310476e-03, -1.305619e-03, -1.316267e-03,
			-1.295867e-03, -1.295676e-03, -1.301257e-03, -1.288990e-03, -1.293333e-03,
		},
		"adjustment": {
			-1.472242e-03, -1.455322e-03, -1.471665e-03, -1.466210e-03, -1.478167e-03,
			-1.455258e-03, -1.455044e-03, -1.461312e-03, -1.447536e-03, -1.452413e-03,
		},
		"initial voltage": {
			-1.053848e-03, -1.038781e-03, -1.053333e-03, -1.048476e-03, -1.059124e-03,
			-1.038724e-03, -1.038533e-03, -1.044114e-03, -1.031848e-03, -1.036190e-03,
		},
		"lead resistance": {
			-1.310990e-03, -1.295924e-03, -1.310476e-03, -1.305619e-03, -1.316267e-03,
			-1.295867e-03, -1.295676e-03, -1.301257e-03, -1.288990e-03, -1.293333e-03,
		},
		"all": {
			-1.183471e-03, -1.166551e-03, -1.182893e-03, -1.177439e-03, -1.189396e-03,
			-1.166487e-03, -1.166273e-03, -1.172540e-03, -1.158765e-03, -1.163642e-03,
		},
	},
	StrainScalerConfigFullBridge2: {
		"baseline": {
			-2.016908e-03, -1.993729e-03, -2.016117e-03, -2.008645e-03, -2.025026e-03,
			-1.993641e-03, -1.993348e-03, -2.001934e-03, -1.983062e-03, -1.989744e-03,
		},
		"adjustment": {
			-2.264988e-03, -2.238958e-03, -2.264100e-03, -2.255708e-03, -2.274104e-03,
			-2.238859e-03, -2.238530e-03, -2.248172e-03, -2.226979e-03, -2.234482e-03,
		},
		"initial voltage": {
			-1.621304e-03, -1.598125e-03, -1.620513e-03, -1.613040e-03, -1.629421e-03,
			-1.598037e-03, -1.597744e-03, -1.606330e-03, -1.587458e-03, -1.594139e-03,
		},
		"lead resistance": {
			-2.016908e-03, -1.993729e-03, -2.016117e-03, -2.008645e-03, -2.025026e-03,
			-1.993641e-03, -1.993348e-03, -2.001934e-03, -1.983062e-03, -1.989744e-03,
		},
		"all": {
			-1.820724e-03, -1.794694e-03, -1.819836e-03, -1.811444e-03, -1.829840e-03,
			-1.794595e-03, -1.794266e-03, -1.803908e-03, -1.782715e-03, -1.790218e-03,
		},
	},
	StrainScalerConfigFullBridge3: {
		"baseline": {
			-2.013923e-03, -1.990812e-03, -2.013134e-03, -2.005684e-03, -2.022016e-03,
			-1.990724e-03, -1.990432e-03, -1.998993e-03, -1.980176e-03, -1.986838e-03,
		},
		"adjustment": {
			-2.261635e-03, -2.235681e-03, -2.260750e-03, -2.252383e-03, -2.270724e-03,
			-2.235583e-03, -2.235255e-03, -2.244869e-03, -2.223738e-03, -2.231219e-03,
		},
		"initial voltage": {
			-1.619374e-03, -1.596250e-03, -1.618585e-03, -1.611130e-03, -1.627472e-03,
			-1.596162e-03, -1.595869e-03, -1.604435e-03, -1.585608e-03, -1.592274e-03,
		},
		"lead resistance": {
			-2.013923e-03, -1.990812e-03, -2.013134e-03, -2.005684e-03, -2.022016e-03,
			-1.990724e-03, -1.990432e-03, -1.998993e-03, -1.980176e-03, -1.986838e-03,
		},
		"all": {
			-1.818557e-03, -1.792588e-03, -1.817671e-03, -1.809299e-03, -1.827651e-03,
			-1.792490e-03, -1.792161e-03, -1.801781e-03, -1.780638e-03, -1.788123e-03,
		},
	},
	StrainScalerConfigHalfBridge1: {
		"baseline": {
			-4.021893e-03, -3.975806e-03, -4.020319e-03, -4.005462e-03, -4.038031e-03,
			-3.975631e-03, -3.975048e-03, -3.992120e-03, -3.954596e-03, -3.967881e-03,
		},
		"adjustment": {
			-4.516585e-03, -4.464830e-03, -4.514819e-03, -4.498134e-03, -4.534709e-03,
			-4.464633e-03, -4.463979e-03, -4.483151e-03, -4.441012e-03, -4.455931e-03,
		},
		"initial voltage": {
			-3.234898e-03, -3.188758e-03, -3.233323e-03, -3.218449e-03, -3.251055e-03,
			-3.188583e-03, -3.188000e-03, -3.205091e-03, -3.167524e-03, -3.180824e-03,
		},
		"lead resistance": {
			-4.036073e-03, -3.989823e-03, -4.034494e-03, -4.019585e-03, -4.052268e-03,
			-3.989648e-03, -3.989063e-03, -4.006195e-03, -3.968539e-03, -3.981871e-03,
		},
		"all": {
			-3.645599e-03, -3.593601e-03, -3.643824e-03, -3.627061e-03, -3.663807e-03,
			-3.593403e-03, -3.592746e-03, -3.612008e-03, -3.569671e-03, -3.584660e-03,
		},
	},
	StrainScalerConfigHalfBridge2: {
		"baseline": {
			-2.621981e-03, -2.591848e-03, -2.620952e-03, -2.611238e-03, -2.632533e-03,
			-2.591733e-03, -2.591352e-03, -2.602514e-03, -2.577981e-03, -2.586667e-03,
		},
		"adjustment": {
			-2.944485e-03, -2.910645e-03, -2.943330e-03, -2.932420e-03, -2.956335e-03,
			-2.910517e-03, -2.910089e-03, -2.922624e-03, -2.895073e-03, -2.904827e-03,
		},
		"initial voltage": {
			-2.107695e-03, -2.077562e-03, -2.106667e-03, -2.096952e-03, -2.118248e-03,
			-2.077448e-03, -2.077067e-03, -2.088229e-03, -2.063695e-03, -2.072381e-03,
		},
		"lead resistance": {
			-2.631225e-03, -2.600986e-03, -2.630193e-03, -2.620445e-03, -2.641815e-03,
			-2.600871e-03, -2.600489e-03, -2.611690e-03, -2.587070e-03, -2.595787e-03,
		},
		"all": {
			-2.375287e-03, -2.341328e-03, -2.374128e-03, -2.363180e-03, -2.387179e-03,
			-2.341199e-03, -2.340770e-03, -2.353349e-03, -2.325701e-03, -2.335489e-03,
		},
	},
	StrainScalerConfigQuarterBridge1: {
		"baseline": {
			-5.215246e-03, -5.155634e-03, -5.213211e-03, -5.193994e-03, -5.236120e-03,
			-5.155408e-03, -5.154654e-03, -5.176736e-03, -5.128199e-03, -5.145384e-03,
		},
		"adjustment": {
			-5.856721e-03, -5.789777e-03, -5.854436e-03, -5.832856e-03, -5.880162e-03,
			-5.789523e-03, -5.788676e-03, -5.813475e-03, -5.758968e-03, -5.778266e-03,
		},
		"initial voltage": {
			-4.196815e-03, -4.137074e-03, -4.194776e-03, -4.175517e-03, -4.217733e-03,
			-4.136848e-03, -4.136092e-03, -4.158222e-03, -4.109581e-03, -4.126802e-03,
		},
		"lead resistance": {
			-5.233633e-03, -5.173811e-03, -5.231592e-03, -5.212307e-03, -5.254581e-03,
			-5.173584e-03, -5.172828e-03, -5.194988e-03, -5.146280e-03, -5.163525e-03,
		},
		"all": {
			-4.729640e-03, -4.662315e-03, -4.727342e-03, -4.705639e-03, -4.753214e-03,
			-4.662059e-03, -4.661208e-03, -4.686147e-03, -4.631330e-03, -4.650738e-03,
		},
	},
	StrainScalerConfigQuarterBridge2: {
		"baseline": {
			-5.215246e-03, -5.155634e-03, -5.213211e-03, -5.193994e-03, -5.236120e-03,
			-5.155408e-03, -5.154654e-03, -5.176736e-03, -5.128199e-03, -5.145384e-03,
		},
		"adjustment": {
			-5.856721e-03, -5.789777e-03, -5.854436e-03, -5.832856e-03, -5.880162e-03,
			-5.789523e-03, -5.788676e-03, -5.813475e-03, -5.758968e-03, -5.778266e-03,
		},
		"initial voltage": {
			-4.196815e-03, -4.137074e-03, -4.194776e-03, -4.175517e-03, -4.217733e-03,
			-4.136848e-03, -4.136092e-03, -4.158222e-03, -4.109581e-03, -4.126802e-03,
		},
		"lead resistance": {
			-5.233633e-03, -5.173811e-03, -5.231592e-03, -5.212307e-03, -5.254581e-03,
			-5.173584e-03, -5.172828e-03, -5.194988e-03, -5.146280e-03, -5.163525e-03,
		},
		"all": {
			-4.729640e-03, -4.662315e-03, -4.727342e-03, -4.705639e-03, -4.753214e-03,
			-4.662059e-03, -4.661208e-03, -4.686147e-03, -4.631330e-03, -4.650738e-03,
		},
	},
}

func TestStrainScaler(t *testing.T) {
	configs := []StrainScalerConfig{
		StrainScalerConfigFullBridge1,
		StrainScalerConfigFullBridge2,
		StrainScalerConfigFullBridge3,
		StrainScalerConfigHalfBridge1,
		StrainScalerConfigHalfBridge2,
		StrainScalerConfigQuarterBridge1,
		StrainScalerConfigQuarterBridge2,
	}
	features := []string{"baseline", "adjustment", "initial voltage", "lead resistance", "all"}

	for _, config := range configs {
		for _, feature := range features {
			t.Run(fmt.Sprintf("config %d / %s", int(config), feature), func(t *testing.T) {
				leadWireResistance := 0.0
				initialBridgeVoltage := 0.0
				gainAdjustment := 1.0

				if feature == "adjustment" || feature == "all" {
					gainAdjustment = 1.123
				}
				if feature == "initial voltage" || feature == "all" {
					initialBridgeVoltage = 0.00135
				}
				if feature == "lead resistance" || feature == "all" {
					leadWireResistance = 1.234
				}

				props := NewProperties().
					Add("NI_Number_Of_Scales", 1).
					Add("NI_Scale[0]_Scale_Type", "Strain").
					Add("NI_Scale[0]_Strain_Configuration", int(config)).
					Add("NI_Scale[0]_Strain_Poisson_Ratio", 0.3).
					Add("NI_Scale[0]_Strain_Gage_Resistance", 350.0).
					Add("NI_Scale[0]_Strain_Lead_Wire_Resistance", leadWireResistance).
					Add("NI_Scale[0]_Strain_Initial_Bridge_Voltage", initialBridgeVoltage).
					Add("NI_Scale[0]_Strain_Gage_Factor", 2.1).
					Add("NI_Scale[0]_Strain_Bridge_Shunt_Calibration_Gain_Adjustment", gainAdjustment).
					Add("NI_Scale[0]_Strain_Voltage_Excitation", 2.5).
					Add("NI_Scale[0]_Strain_Input_Source", scaleIndexRawDataInput)

				scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
				if err != nil {
					t.Fatalf("Error creating scaler: %v", err)
				}

				checkScaler(t, scaler, []Scaler{&StrainScaler{}})

				input := []float64{
					0.0068827, 0.0068036, 0.0068800, 0.0068545, 0.0069104,
					0.0068033, 0.0068023, 0.0068316, 0.0067672, 0.0067900,
				}

				if err := scaler.Allocate(len(input)); err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}

				got, err := scaler.Scale(input)
				if err != nil {
					t.Fatalf("Error scaling input: %v", err)
				}

				if _, ok := got.([]float64); !ok {
					t.Fatalf("Expected output to be a slice of float64, got %T", got)
				}

				want := wantStrainOutput[config][feature]
				if !cmp.Equal(want, got, floatOpt) {
					t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
				}
			})
		}
	}
}

func TestRTDUnsupportedConfig(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "Strain").
		Add("NI_Scale[0]_Strain_Configuration", 12345). // Not a valid config
		Add("NI_Scale[0]_Strain_Poisson_Ratio", 0.3).
		Add("NI_Scale[0]_Strain_Gage_Resistance", 350.0).
		Add("NI_Scale[0]_Strain_Lead_Wire_Resistance", 0.0).
		Add("NI_Scale[0]_Strain_Initial_Bridge_Voltage", 0.0).
		Add("NI_Scale[0]_Strain_Gage_Factor", 2.1).
		Add("NI_Scale[0]_Strain_Bridge_Shunt_Calibration_Gain_Adjustment", 0.0).
		Add("NI_Scale[0]_Strain_Voltage_Excitation", 2.5).
		Add("NI_Scale[0]_Strain_Input_Source", scaleIndexRawDataInput)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Error creating scaler: %v", err)
	}

	checkScaler(t, scaler, []Scaler{&StrainScaler{}})

	input := []float64{1, 2, 3}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	output, err := scaler.Scale(input)
	if !errors.Is(err, ErrUnsupportedStrainConfiguration) {
		t.Fatalf("Expected error to be ErrUnsupportedStrainConfiguration, got %v", err)
	} else if err == nil {
		t.Fatalf("Expected error, got nil")
	}

	if output != nil {
		t.Fatalf("Expected output to be nil, got %v", output)
	}
}

func TestTableScaler(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "Table").
		Add("NI_Scale[0]_Table_Scaled_Values_Size", 3).
		Add("NI_Scale[0]_Table_Scaled_Values[0]", 1.0).
		Add("NI_Scale[0]_Table_Scaled_Values[1]", 2.0).
		Add("NI_Scale[0]_Table_Scaled_Values[2]", 3.0).
		Add("NI_Scale[0]_Table_Pre_Scaled_Values_Size", 3).
		Add("NI_Scale[0]_Table_Pre_Scaled_Values[0]", 2.0).
		Add("NI_Scale[0]_Table_Pre_Scaled_Values[1]", 4.0).
		Add("NI_Scale[0]_Table_Pre_Scaled_Values[2]", 8.0)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Error creating scaler: %v", err)
	}

	checkScaler(t, scaler, []Scaler{&TableScaler{}})

	input := []float64{0.5, 1, 1.5, 2.5, 3, 3.5}
	want := []float64{2, 2, 3, 6, 8, 8}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Error scaling input: %v", err)
	}

	if _, ok := got.([]float64); !ok {
		t.Fatalf("Expected output to be a slice of float64, got %T", got)
	}

	if !cmp.Equal(want, got, floatOpt) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
	}
}

func TestThermistorScalerVoltageExcitation(t *testing.T) {
	cases := []struct {
		name                    string
		resistanceConfiguration int
		leadResistance          float64
		want                    []float64
	}{
		{"2-wire, no lead resistance", 2, 0.0, []float64{287.1495569816, 290.71633623, 294.4862276706}},
		{"2-wire, w/ lead resistance", 2, 100.0, []float64{287.1495569816, 290.71633623, 294.4862276706}},
		{"3-wire, no lead resistance", 3, 0.0, []float64{287.1495569816, 290.71633623, 294.4862276706}},
		{"3-wire, w/ lead resistance", 3, 100.0, []float64{287.4248927942, 291.0482875767, 294.8892119392}},
		{"4-wire, no lead resistance", 4, 0.0, []float64{287.1495569816, 290.71633623, 294.4862276706}},
		{"4-wire, w/ lead resistance", 4, 100.0, []float64{287.1495569816, 290.71633623, 294.4862276706}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := NewProperties().
				Add("NI_Number_Of_Scales", 1).
				Add("NI_Scale[0]_Scale_Type", "Thermistor").
				Add("NI_Scale[0]_Thermistor_Resistance_Configuration", tc.resistanceConfiguration).
				Add("NI_Scale[0]_Thermistor_Excitation_Type", int(excitationTypeVoltage)).
				Add("NI_Scale[0]_Thermistor_Excitation_Value", 2.5).
				Add("NI_Scale[0]_Thermistor_R1_Reference_Resistance", 10000.0).
				Add("NI_Scale[0]_Thermistor_Lead_Wire_Resistance", tc.leadResistance).
				Add("NI_Scale[0]_Thermistor_A", 0.0012873851).
				Add("NI_Scale[0]_Thermistor_B", 0.00023575235).
				Add("NI_Scale[0]_Thermistor_C", 9.497806e-8).
				Add("NI_Scale[0]_Thermistor_Temperature_Offset", 1.0).
				Add("NI_Scale[0]_Thermistor_Input_Source", scaleIndexRawDataInput)

			scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
			if err != nil {
				t.Fatalf("Error creating scaler: %v", err)
			}

			checkScaler(t, scaler, []Scaler{&ThermistorScaler{}})

			input := []float64{1.1, 1.0, 0.9}

			if err := scaler.Allocate(len(input)); err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			got, err := scaler.Scale(input)
			if err != nil {
				t.Fatalf("Error scaling input: %v", err)
			}

			if _, ok := got.([]float64); !ok {
				t.Fatalf("Expected output to be a slice of float64, got %T", got)
			}

			if !cmp.Equal(tc.want, got, floatOpt) {
				t.Fatalf("Output differs: %v", cmp.Diff(tc.want, got, floatOpt))
			}
		})
	}
}

func TestThermistorScalerCurrentExcitation(t *testing.T) {
	cases := []struct {
		name                    string
		resistanceConfiguration int
		leadResistance          float64
		want                    []float64
	}{
		{"2-wire, no lead resistance", 2, 0.0, []float64{335.5876272527, 338.303823856, 341.3530400858}},
		{"2-wire, w/ lead resistance", 2, 100.0, []float64{341.3530400858, 344.8212218133, 348.831282405}},
		{"3-wire, no lead resistance", 3, 0.0, []float64{335.5876272527, 338.303823856, 341.3530400858}},
		{"3-wire, w/ lead resistance", 3, 100.0, []float64{338.303823856, 341.3530400858, 344.8212218133}},
		{"4-wire, no lead resistance", 4, 0.0, []float64{335.5876272527, 338.303823856, 341.3530400858}},
		{"4-wire, w/ lead resistance", 4, 100.0, []float64{335.5876272527, 338.303823856, 341.3530400858}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := NewProperties().
				Add("NI_Number_Of_Scales", 1).
				Add("NI_Scale[0]_Scale_Type", "Thermistor").
				Add("NI_Scale[0]_Thermistor_Resistance_Configuration", tc.resistanceConfiguration).
				Add("NI_Scale[0]_Thermistor_Excitation_Type", int(excitationTypeCurrent)).
				Add("NI_Scale[0]_Thermistor_Excitation_Value", 1.0e-3).
				Add("NI_Scale[0]_Thermistor_R1_Reference_Resistance", 0.0).
				Add("NI_Scale[0]_Thermistor_Lead_Wire_Resistance", tc.leadResistance).
				Add("NI_Scale[0]_Thermistor_A", 0.0012873851).
				Add("NI_Scale[0]_Thermistor_B", 0.00023575235).
				Add("NI_Scale[0]_Thermistor_C", 9.497806e-8).
				Add("NI_Scale[0]_Thermistor_Temperature_Offset", 1.0).
				Add("NI_Scale[0]_Thermistor_Input_Source", scaleIndexRawDataInput)

			scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
			if err != nil {
				t.Fatalf("Error creating scaler: %v", err)
			}

			checkScaler(t, scaler, []Scaler{&ThermistorScaler{}})

			input := []float64{1.1, 1.0, 0.9}

			if err := scaler.Allocate(len(input)); err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			got, err := scaler.Scale(input)
			if err != nil {
				t.Fatalf("Error scaling input: %v", err)
			}

			if _, ok := got.([]float64); !ok {
				t.Fatalf("Expected output to be a slice of float64, got %T", got)
			}

			if !cmp.Equal(tc.want, got, floatOpt) {
				t.Fatalf("Output differs: %v", cmp.Diff(tc.want, got, floatOpt))
			}
		})
	}
}

func TestThermistorUnsupportedExcitationType(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "Thermistor").
		Add("NI_Scale[0]_Thermistor_Resistance_Configuration", 3).
		Add("NI_Scale[0]_Thermistor_Excitation_Type", 12345). // Not a valid excitation type
		Add("NI_Scale[0]_Thermistor_Excitation_Value", 2.5).
		Add("NI_Scale[0]_Thermistor_R1_Reference_Resistance", 10000.0).
		Add("NI_Scale[0]_Thermistor_Lead_Wire_Resistance", 0.0).
		Add("NI_Scale[0]_Thermistor_A", 0.0012873851).
		Add("NI_Scale[0]_Thermistor_B", 0.00023575235).
		Add("NI_Scale[0]_Thermistor_C", 9.497806e-8).
		Add("NI_Scale[0]_Thermistor_Temperature_Offset", 1.0).
		Add("NI_Scale[0]_Thermistor_Input_Source", scaleIndexRawDataInput)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Error creating scaler: %v", err)
	}

	checkScaler(t, scaler, []Scaler{&ThermistorScaler{}})

	input := []float64{1.1, 1.0, 0.9}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err == nil {
		t.Fatalf("Expected ErrUnsupportedExcitationType, got nil")
	}
	if got != nil {
		t.Fatalf("Expected output to be nil, got %v", got)
	}
}

func TestThermocoupleScalerVoltageToTemperature(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "Thermocouple").
		Add("NI_Scale[0]_Thermocouple_Thermocouple_Type", int(thermocoupleTypeK)).
		Add("NI_Scale[0]_Thermocouple_Scaling_Direction", 0).
		Add("NI_Scale[0]_Thermocouple_Input_Source", scaleIndexRawDataInput)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Error creating scaler: %v", err)
	}

	checkScaler(t, scaler, []Scaler{&ThermocoupleScaler{}})

	input := []float64{0, 10, 100, 1000}
	want := []float64{0, 0.250843110, 2.50889889, 24.9836476}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Error scaling input: %v", err)
	}

	if _, ok := got.([]float64); !ok {
		t.Fatalf("Expected output to be a slice of float64, got %T", got)
	}

	if !cmp.Equal(want, got, cmpopts.EquateApprox(1e-7, 1e-7)) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, cmpopts.EquateApprox(1e-7, 1e-7)))
	}
}

func TestThermocoupleScalerTemperatureToVoltage(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 1).
		Add("NI_Scale[0]_Scale_Type", "Thermocouple").
		Add("NI_Scale[0]_Thermocouple_Thermocouple_Type", int(thermocoupleTypeK)).
		Add("NI_Scale[0]_Thermocouple_Scaling_Direction", 1).
		Add("NI_Scale[0]_Thermocouple_Input_Source", scaleIndexRawDataInput)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Error creating scaler: %v", err)
	}

	checkScaler(t, scaler, []Scaler{&ThermocoupleScaler{}})

	input := []float64{0, 10, 50, 100}
	want := []float64{0, 396.8619078, 2023.0778862, 4096.2302187}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Error scaling input: %v", err)
	}

	if _, ok := got.([]float64); !ok {
		t.Fatalf("Expected output to be a slice of float64, got %T", got)
	}

	// There's a massive error in the first value here but I've checked and the
	// exact same error is present in npTDMS as well.
	if !cmp.Equal(want, got, cmpopts.EquateApprox(1e-4, 1e-4)) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, cmpopts.EquateApprox(1e-4, 1e-4)))
	}
}

func TestAddScaler(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 2).
		Add("NI_Scale[0]_Scale_Type", "Linear").
		Add("NI_Scale[0]_Linear_Slope", 2.0).
		Add("NI_Scale[0]_Linear_Y_Intercept", 10.0).
		Add("NI_Scale[1]_Scale_Type", "Add").
		Add("NI_Scale[1]_Add_Left_Operand_Input_Source", scaleIndexRawDataInput).
		Add("NI_Scale[1]_Add_Right_Operand_Input_Source", 0)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeInt32)
	if err != nil {
		t.Fatalf("Error getting object scaler: %v", err)
	}

	checkScaler(t, scaler, []Scaler{&LinearScaler{}, &AddScaler{}})

	input := []int32{1, 2, 3}
	want := []float64{13, 16, 19}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Error scaling input: %v", err)
	}
	if !cmp.Equal(want, got, floatOpt) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
	}
}

func TestSubtractScaler(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 2).
		Add("NI_Number_Of_Scales", 2).
		Add("NI_Scale[0]_Scale_Type", "Linear").
		Add("NI_Scale[0]_Linear_Slope", 2.0).
		Add("NI_Scale[0]_Linear_Y_Intercept", 10.0).
		Add("NI_Scale[1]_Scale_Type", "Subtract").
		Add("NI_Scale[1]_Subtract_Left_Operand_Input_Source", scaleIndexRawDataInput).
		Add("NI_Scale[1]_Subtract_Right_Operand_Input_Source", 0)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeInt32)
	if err != nil {
		t.Fatalf("Error getting object scaler: %v", err)
	}

	checkScaler(t, scaler, []Scaler{&LinearScaler{}, &SubtractScaler{}})

	input := []int32{1, 2, 3}
	want := []float64{-11, -12, -13}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Error scaling input: %v", err)
	}
	if !cmp.Equal(want, got, floatOpt) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
	}
}

func TestMultipleScalers(t *testing.T) {
	props := NewProperties().
		Add("NI_Number_Of_Scales", 3).
		Add("NI_Scaling_Status", "unscaled").
		Add("NI_Scale[0]_Scale_Type", "Linear").
		Add("NI_Scale[0]_Linear_Slope", 1.0).
		Add("NI_Scale[0]_Linear_Y_Intercept", 1.0).
		Add("NI_Scale[0]_Linear_Input_Source", scaleIndexRawDataInput).
		Add("NI_Scale[1]_Scale_Type", "Linear").
		Add("NI_Scale[1]_Linear_Slope", 2.0).
		Add("NI_Scale[1]_Linear_Y_Intercept", 2.0).
		Add("NI_Scale[1]_Linear_Input_Source", 0).
		Add("NI_Scale[2]_Scale_Type", "Linear").
		Add("NI_Scale[2]_Linear_Slope", 3.0).
		Add("NI_Scale[2]_Linear_Y_Intercept", 3.0).
		Add("NI_Scale[2]_Linear_Input_Source", 1)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Error getting object scaler: %v", err)
	}

	checkScaler(t, scaler, []Scaler{&LinearScaler{}, &LinearScaler{}, &LinearScaler{}})

	input := []float64{1, 2, 3}
	want := []float64{21, 27, 33}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Error scaling input: %v", err)
	}

	if _, ok := got.([]float64); !ok {
		t.Fatalf("Expected output to be a slice of float64, got %T", got)
	}

	if !cmp.Equal(want, got, floatOpt) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
	}
}

func TestMultipleScalerWithAllRawDataInput(t *testing.T) {
	// When multiple scalers have raw data input, latest one overwrites data.

	props := NewProperties().
		Add("NI_Number_Of_Scales", 3).
		Add("NI_Scaling_Status", "unscaled").
		Add("NI_Scale[0]_Scale_Type", "Linear").
		Add("NI_Scale[0]_Linear_Slope", 1.0).
		Add("NI_Scale[0]_Linear_Y_Intercept", 1.0).
		Add("NI_Scale[0]_Linear_Input_Source", scaleIndexRawDataInput).
		Add("NI_Scale[1]_Scale_Type", "Linear").
		Add("NI_Scale[1]_Linear_Slope", 2.0).
		Add("NI_Scale[1]_Linear_Y_Intercept", 2.0).
		Add("NI_Scale[1]_Linear_Input_Source", scaleIndexRawDataInput).
		Add("NI_Scale[2]_Scale_Type", "Linear").
		Add("NI_Scale[2]_Linear_Slope", 3.0).
		Add("NI_Scale[2]_Linear_Y_Intercept", 3.0).
		Add("NI_Scale[2]_Linear_Input_Source", scaleIndexRawDataInput)

	scaler, err := getObjectScaler(&object{properties: props}, DataTypeFloat64)
	if err != nil {
		t.Fatalf("Error getting object scaler: %v", err)
	}

	checkScaler(t, scaler, []Scaler{&LinearScaler{}, &LinearScaler{}, &LinearScaler{}})

	input := []float64{1, 2, 3}
	want := []float64{6, 9, 12}

	if err := scaler.Allocate(len(input)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, err := scaler.Scale(input)
	if err != nil {
		t.Fatalf("Error scaling input: %v", err)
	}

	if _, ok := got.([]float64); !ok {
		t.Fatalf("Expected output to be a slice of float64, got %T", got)
	}

	if !cmp.Equal(want, got, floatOpt) {
		t.Fatalf("Output differs: %v", cmp.Diff(want, got, floatOpt))
	}
}

func checkScaler(t *testing.T, scaler *Multiscaler, wantScalers []Scaler) {
	if scaler == nil {
		t.Fatal("expected scaler to be non-nil")
	}

	if len(scaler.scalers) != len(wantScalers) {
		t.Fatalf("expected %d scalers, got %d", len(wantScalers), len(scaler.scalers))
	}

	for i, want := range wantScalers {
		wantType := reflect.TypeOf(want).Elem()
		gotType := reflect.TypeOf(scaler.scalers[i]).Elem()

		if gotType != wantType {
			t.Errorf("expected scaler %d to be of type %v, got %v", i, wantType, gotType)
		}
	}
}
