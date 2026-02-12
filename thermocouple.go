package tdms

import "math"

type thermocouple struct {
	forwardPolynomials []polynomial
	inversePolynomials []polynomial
	exponentialTerm    []float64
}

func newThermocouple(forwardPolynomials, inversePolynomials []polynomial, exponentialTerm []float64) thermocouple {
	// Thermocouples are instantiated as "constant" package variables so if
	// they're not valid, we can panic.

	if !isContiguous(forwardPolynomials) {
		panic("thermocouple forward polynomials must be contiguous")
	}

	if !isContiguous(inversePolynomials) {
		panic("thermocouple inverse polynomials must be contiguous")
	}

	return thermocouple{
		forwardPolynomials: forwardPolynomials,
		inversePolynomials: inversePolynomials,
		exponentialTerm:    exponentialTerm,
	}
}

// isContiguous checks whether the input polynomials contiguously covers all
// possible input values without overlapping.
func isContiguous(polynomials []polynomial) bool {
	prevEnd := math.Inf(-1)

	for _, p := range polynomials {
		if p.rangeEnd <= p.rangeStart || p.rangeStart != prevEnd {
			return false
		}

		prevEnd = p.rangeEnd
	}

	// If the final end is infinity, we've covered the whole real line.
	return prevEnd == math.Inf(1)
}

func (t thermocouple) temperatureToVoltage(temperature float64) float64 {
	voltage := math.NaN()

	for _, poly := range t.forwardPolynomials {
		if poly.withinRange(temperature) {
			voltage = poly.apply(temperature)
			break
		}
	}

	// Type K thermocouples have an additional expontential term that applies
	// when temperature is above freezing.
	if t.exponentialTerm == nil || temperature < 0 {
		return voltage
	}

	a0, a1, a2 := t.exponentialTerm[0], t.exponentialTerm[1], t.exponentialTerm[2]
	diff := temperature - a2
	return voltage + a0*math.Exp(a1*diff*diff)
}

func (t thermocouple) voltageToTemperature(voltage float64) float64 {
	temperature := math.NaN()

	for _, poly := range t.inversePolynomials {
		if poly.withinRange(voltage) {
			temperature = poly.apply(voltage)
			break
		}
	}

	return temperature
}

type polynomial struct {
	rangeStart   float64
	rangeEnd     float64
	coefficients []float64
}

func (p polynomial) withinRange(value float64) bool {
	return p.rangeStart <= value && value < p.rangeEnd
}

func (p polynomial) apply(value float64) float64 {
	result := 0.0

	for i, coeff := range p.coefficients {
		result += coeff * math.Pow(value, float64(i))
	}

	return result
}

// All the below values are taken from npTDMS which are in turn taken from NIST
// website with very slight changes to types R and S.
//
// See:
// https://github.com/adamreeve/npTDMS/blob/master/nptdms/thermocouples.py
// https://srdata.nist.gov/its90/main/

var thermocoupleB = newThermocouple(
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   630.615,
			coefficients: []float64{
				0.000000000000e+00,
				-0.246508183460e-03,
				0.590404211710e-05,
				-0.132579316360e-08,
				0.156682919010e-11,
				-0.169445292400e-14,
				0.629903470940e-18,
			},
		},
		{
			rangeStart: 630.615,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				-0.389381686210e+01,
				0.285717474700e-01,
				-0.848851047850e-04,
				0.157852801640e-06,
				-0.168353448640e-09,
				0.111097940130e-12,
				-0.445154310330e-16,
				0.989756408210e-20,
				-0.937913302890e-24,
			},
		},
	},
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   2.431,
			coefficients: []float64{
				9.8423321e+01,
				6.9971500e+02,
				-8.4765304e+02,
				1.0052644e+03,
				-8.3345952e+02,
				4.5508542e+02,
				-1.5523037e+02,
				2.9886750e+01,
				-2.4742860e+00,
			},
		},
		{
			rangeStart: 2.431,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				2.1315071e+02,
				2.8510504e+02,
				-5.2742887e+01,
				9.9160804e+00,
				-1.2965303e+00,
				1.1195870e-01,
				-6.0625199e-03,
				1.8661696e-04,
				-2.4878585e-06,
			},
		},
	},
	nil,
)

var thermocoupleE = newThermocouple(
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   0.000,
			coefficients: []float64{
				0.000000000000e+00,
				0.586655087080e-01,
				0.454109771240e-04,
				-0.779980486860e-06,
				-0.258001608430e-07,
				-0.594525830570e-09,
				-0.932140586670e-11,
				-0.102876055340e-12,
				-0.803701236210e-15,
				-0.439794973910e-17,
				-0.164147763550e-19,
				-0.396736195160e-22,
				-0.558273287210e-25,
				-0.346578420130e-28,
			},
		},
		{
			rangeStart: 0.000,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				0.000000000000e+00,
				0.586655087100e-01,
				0.450322755820e-04,
				0.289084072120e-07,
				-0.330568966520e-09,
				0.650244032700e-12,
				-0.191974955040e-15,
				-0.125366004970e-17,
				0.214892175690e-20,
				-0.143880417820e-23,
				0.359608994810e-27,
			},
		},
	},
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   0.000,
			coefficients: []float64{
				0.0000000e+00,
				1.6977288e+01,
				-4.3514970e-01,
				-1.5859697e-01,
				-9.2502871e-02,
				-2.6084314e-02,
				-4.1360199e-03,
				-3.4034030e-04,
				-1.1564890e-05,
				0.0000000e+00,
			},
		},
		{
			rangeStart: 0.000,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				0.0000000e+00,
				1.7057035e+01,
				-2.3301759e-01,
				6.5435585e-03,
				-7.3562749e-05,
				-1.7896001e-06,
				8.4036165e-08,
				-1.3735879e-09,
				1.0629823e-11,
				-3.2447087e-14,
			},
		},
	},
	nil,
)

var thermocoupleJ = newThermocouple(
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   760.000,
			coefficients: []float64{
				0.000000000000e+00,
				0.503811878150e-01,
				0.304758369300e-04,
				-0.856810657200e-07,
				0.132281952950e-09,
				-0.170529583370e-12,
				0.209480906970e-15,
				-0.125383953360e-18,
				0.156317256970e-22,
			},
		},
		{
			rangeStart: 760.000,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				0.296456256810e+03,
				-0.149761277860e+01,
				0.317871039240e-02,
				-0.318476867010e-05,
				0.157208190040e-08,
				-0.306913690560e-12,
			},
		},
	},
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   0.000,
			coefficients: []float64{
				0.0000000e+00,
				1.9528268e+01,
				-1.2286185e+00,
				-1.0752178e+00,
				-5.9086933e-01,
				-1.7256713e-01,
				-2.8131513e-02,
				-2.3963370e-03,
				-8.3823321e-05,
			},
		},
		{
			rangeStart: 0.000,
			rangeEnd:   42.919,
			coefficients: []float64{
				0.000000e+00,
				1.978425e+01,
				-2.001204e-01,
				1.036969e-02,
				-2.549687e-04,
				3.585153e-06,
				-5.344285e-08,
				5.099890e-10,
				0.000000e+00,
			},
		},
		{
			rangeStart: 42.919,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				-3.11358187e+03,
				3.00543684e+02,
				-9.94773230e+00,
				1.70276630e-01,
				-1.43033468e-03,
				4.73886084e-06,
				0.00000000e+00,
				0.00000000e+00,
				0.00000000e+00,
			},
		},
	},
	nil,
)

var thermocoupleK = newThermocouple(
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   0.000,
			coefficients: []float64{
				0.000000000000e+00,
				0.394501280250e-01,
				0.236223735980e-04,
				-0.328589067840e-06,
				-0.499048287770e-08,
				-0.675090591730e-10,
				-0.574103274280e-12,
				-0.310888728940e-14,
				-0.104516093650e-16,
				-0.198892668780e-19,
				-0.163226974860e-22,
			},
		},
		{
			rangeStart: 0.000,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				-0.176004136860e-01,
				0.389212049750e-01,
				0.185587700320e-04,
				-0.994575928740e-07,
				0.318409457190e-09,
				-0.560728448890e-12,
				0.560750590590e-15,
				-0.320207200030e-18,
				0.971511471520e-22,
				-0.121047212750e-25,
			},
		},
	},
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   0.000,
			coefficients: []float64{
				0.0000000e+00,
				2.5173462e+01,
				-1.1662878e+00,
				-1.0833638e+00,
				-8.9773540e-01,
				-3.7342377e-01,
				-8.6632643e-02,
				-1.0450598e-02,
				-5.1920577e-04,
				0.0000000e+00,
			},
		},
		{
			rangeStart: 0.000,
			rangeEnd:   20.644,
			coefficients: []float64{
				0.000000e+00,
				2.508355e+01,
				7.860106e-02,
				-2.503131e-01,
				8.315270e-02,
				-1.228034e-02,
				9.804036e-04,
				-4.413030e-05,
				1.057734e-06,
				-1.052755e-08,
			},
		},
		{
			rangeStart: 20.644,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				-1.318058e+02,
				4.830222e+01,
				-1.646031e+00,
				5.464731e-02,
				-9.650715e-04,
				8.802193e-06,
				-3.110810e-08,
				0.000000e+00,
				0.000000e+00,
				0.000000e+00,
			},
		},
	},
	[]float64{
		0.118597600000e+00,
		-0.118343200000e-03,
		0.126968600000e+03,
	},
)

var thermocoupleN = newThermocouple(
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   0.000,
			coefficients: []float64{
				0.000000000000e+00,
				0.261591059620e-01,
				0.109574842280e-04,
				-0.938411115540e-07,
				-0.464120397590e-10,
				-0.263033577160e-11,
				-0.226534380030e-13,
				-0.760893007910e-16,
				-0.934196678350e-19,
			},
		},
		{
			rangeStart: 0.0,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				0.000000000000e+00,
				0.259293946010e-01,
				0.157101418800e-04,
				0.438256272370e-07,
				-0.252611697940e-09,
				0.643118193390e-12,
				-0.100634715190e-14,
				0.997453389920e-18,
				-0.608632456070e-21,
				0.208492293390e-24,
				-0.306821961510e-28,
			},
		},
	},
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   0.000,
			coefficients: []float64{
				0.0000000e+00,
				3.8436847e+01,
				1.1010485e+00,
				5.2229312e+00,
				7.2060525e+00,
				5.8488586e+00,
				2.7754916e+00,
				7.7075166e-01,
				1.1582665e-01,
				7.3138868e-03,
			},
		},
		{
			rangeStart: 0.000,
			rangeEnd:   20.613,
			coefficients: []float64{
				0.00000e+00,
				3.86896e+01,
				-1.08267e+00,
				4.70205e-02,
				-2.12169e-06,
				-1.17272e-04,
				5.39280e-06,
				-7.98156e-08,
				0.00000e+00,
				0.00000e+00,
			},
		},
		{
			rangeStart: 20.613,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				1.972485e+01,
				3.300943e+01,
				-3.915159e-01,
				9.855391e-03,
				-1.274371e-04,
				7.767022e-07,
				0.000000e+00,
				0.000000e+00,
				0.000000e+00,
				0.000000e+00,
			},
		},
	},
	nil,
)

var thermocoupleR = newThermocouple(
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   1064.180,
			coefficients: []float64{
				0.000000000000e+00,
				0.528961729765e-02,
				0.139166589782e-04,
				-0.238855693017e-07,
				0.356916001063e-10,
				-0.462347666298e-13,
				0.500777441034e-16,
				-0.373105886191e-19,
				0.157716482367e-22,
				-0.281038625251e-26,
			},
		},
		{
			rangeStart: 1064.180,
			rangeEnd:   1664.5,
			coefficients: []float64{
				0.295157925316e+01,
				-0.252061251332e-02,
				0.159564501865e-04,
				-0.764085947576e-08,
				0.205305291024e-11,
				-0.293359668173e-15,
			},
		},
		{
			rangeStart: 1664.5,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				0.152232118209e+03,
				-0.268819888545e+00,
				0.171280280471e-03,
				-0.345895706453e-07,
				-0.934633971046e-14,
			},
		},
	},
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   1.923,
			coefficients: []float64{
				0.0000000e+00,
				1.8891380e+02,
				-9.3835290e+01,
				1.3068619e+02,
				-2.2703580e+02,
				3.5145659e+02,
				-3.8953900e+02,
				2.8239471e+02,
				-1.2607281e+02,
				3.1353611e+01,
				-3.3187769e+00,
			},
		},
		{
			rangeStart: 1.923,
			rangeEnd:   11.361,
			coefficients: []float64{
				1.334584505e+01,
				1.472644573e+02,
				-1.844024844e+01,
				4.031129726e+00,
				-6.249428360e-01,
				6.468412046e-02,
				-4.458750426e-03,
				1.994710149e-04,
				-5.313401790e-06,
				6.481976217e-08,
				0.000000000e+00,
			},
		},
		{
			rangeStart: 11.361,
			rangeEnd:   19.739,
			coefficients: []float64{
				-8.199599416e+01,
				1.553962042e+02,
				-8.342197663e+00,
				4.279433549e-01,
				-1.191577910e-02,
				1.492290091e-04,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
			},
		},
		{
			rangeStart: 19.739,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				3.406177836e+04,
				-7.023729171e+03,
				5.582903813e+02,
				-1.952394635e+01,
				2.560740231e-01,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
			},
		},
	},
	nil,
)

var thermocoupleS = newThermocouple(
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   1064.180,
			coefficients: []float64{
				0.000000000000e+00,
				0.540313308631e-02,
				0.125934289740e-04,
				-0.232477968689e-07,
				0.322028823036e-10,
				-0.331465196389e-13,
				0.255744251786e-16,
				-0.125068871393e-19,
				0.271443176145e-23,
			},
		},
		{
			rangeStart: 1064.180,
			rangeEnd:   1664.500,
			coefficients: []float64{
				0.132900444085e+01,
				0.334509311344e-02,
				0.654805192818e-05,
				-0.164856259209e-08,
				0.129989605174e-13,
			},
		},
		{
			rangeStart: 1664.500,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				0.146628232636e+03,
				-0.258430516752e+00,
				0.163693574641e-03,
				-0.330439046987e-07,
				-0.943223690612e-14,
			},
		},
	},
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   1.874,
			coefficients: []float64{
				0.00000000e+00,
				1.84949460e+02,
				-8.00504062e+01,
				1.02237430e+02,
				-1.52248592e+02,
				1.88821343e+02,
				-1.59085941e+02,
				8.23027880e+01,
				-2.34181944e+01,
				2.79786260e+00,
			},
		},
		{
			rangeStart: 1.874,
			rangeEnd:   10.332,
			coefficients: []float64{
				1.291507177e+01,
				1.466298863e+02,
				-1.534713402e+01,
				3.145945973e+00,
				-4.163257839e-01,
				3.187963771e-02,
				-1.291637500e-03,
				2.183475087e-05,
				-1.447379511e-07,
				8.211272125e-09,
			},
		},
		{
			rangeStart: 10.332,
			rangeEnd:   17.536,
			coefficients: []float64{
				-8.087801117e+01,
				1.621573104e+02,
				-8.536869453e+00,
				4.719686976e-01,
				-1.441693666e-02,
				2.081618890e-04,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
			},
		},
		{
			rangeStart: 17.536,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				5.333875126e+04,
				-1.235892298e+04,
				1.092657613e+03,
				-4.265693686e+01,
				6.247205420e-01,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
				0.000000000e+00,
			},
		},
	},
	nil,
)

var thermocoupleT = newThermocouple(
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   0.000,
			coefficients: []float64{
				0.000000000000e+00,
				0.387481063640e-01,
				0.441944343470e-04,
				0.118443231050e-06,
				0.200329735540e-07,
				0.901380195590e-09,
				0.226511565930e-10,
				0.360711542050e-12,
				0.384939398830e-14,
				0.282135219250e-16,
				0.142515947790e-18,
				0.487686622860e-21,
				0.107955392700e-23,
				0.139450270620e-26,
				0.797951539270e-30,
			},
		},
		{
			rangeStart: 0.000,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				0.000000000000e+00,
				0.387481063640e-01,
				0.332922278800e-04,
				0.206182434040e-06,
				-0.218822568460e-08,
				0.109968809280e-10,
				-0.308157587720e-13,
				0.454791352900e-16,
				-0.275129016730e-19,
			},
		},
	},
	[]polynomial{
		{
			rangeStart: math.Inf(-1),
			rangeEnd:   0.000,
			coefficients: []float64{
				0.0000000e+00,
				2.5949192e+01,
				-2.1316967e-01,
				7.9018692e-01,
				4.2527777e-01,
				1.3304473e-01,
				2.0241446e-02,
				1.2668171e-03,
			},
		},
		{
			rangeStart: 0.000,
			rangeEnd:   math.Inf(1),
			coefficients: []float64{
				0.000000e+00,
				2.592800e+01,
				-7.602961e-01,
				4.637791e-02,
				-2.165394e-03,
				6.048144e-05,
				-7.293422e-07,
				0.000000e+00,
			},
		},
	},
	nil,
)
