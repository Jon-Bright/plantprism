package device

import (
	"time"

	"go.einride.tech/pid"
)

const (
	ECRefTemp         = 25.0
	ECFactorPerDegree = 0.0235
	ECSmoothing       = 0.75

	// Values tweaked experimentally. The scales look weird
	// because the controller code is counting seconds (expecting
	// samples at least every few seconds) and our sample diffs
	// are a thousand to tens of thousands of seconds apart. With
	// these values, we take in EC (range 1000-1600ish) and output
	// ml of nutrient (range 15-120).
	ECPropGain = 0.05
	ECInteGain = 0.0000002
	ECDeriGain = 150

	// This is a little bit below what we see after a cleaning
	// cycle and refill with 120ml. It seems like a reasonable
	// midpoint.
	ECGoalValue = 1425
)

func newPIDController() *pid.Controller {
	return &pid.Controller{
		Config: pid.ControllerConfig{
			ProportionalGain: ECPropGain,
			IntegralGain:     ECInteGain,
			DerivativeGain:   ECDeriGain,
		},
	}
}

func (d *Device) updateSmoothedEC(ec int, tempTank float64, lastUpdate time.Time, thisUpdate time.Time) {
	// First, we compensate for temperature. It looks like some
	// kind of EC compensation has already happened (and there's a
	// bunch of lookup tables in the STM32 code that are doing
	// something that might be compensation) - but it's probably
	// corrected _too much_. What we _should_ see: EC should be
	// higher when temperature is higher. What we _actually_ see:
	// EC is lower when temperature is higher.
	//
	// The correction here is a result of a bunch of experiments
	// to find the best output, on the assumption that EC should
	// usually go down between readings. See the sheet linked from
	// the docs for the experiments.
	tempCorrectedEC := float64(ec) / (1.0 - ECFactorPerDegree*(tempTank-ECRefTemp))

	// Next, we smooth the values by integrating some part of the
	// new value with some part of the previous smoothed value, if
	// the smoothed value has ever been set.
	if d.SmoothedEC == 0.0 {
		d.SmoothedEC = tempCorrectedEC
	} else {
		d.SmoothedEC = d.SmoothedEC*ECSmoothing + tempCorrectedEC*(1.0-ECSmoothing)
	}

	// Then, we give the new value to the PID controller
	d.NutrientPID.Update(pid.ControllerInput{
		ReferenceSignal:  ECGoalValue,
		ActualSignal:     d.SmoothedEC,
		SamplingInterval: thisUpdate.Sub(lastUpdate),
	})

	// ...and see if it wants nutrient (and more nutrient than we
	// wanted before).
	wantNutrient := int(d.NutrientPID.State.ControlSignal/5.0) * 5
	log.Info.Printf("Got EC %d, Temp-Corrected %.1f, Smoothed %.1f, ControlSignal %.2f, wantNutrient %d, prev %d", ec, tempCorrectedEC, d.SmoothedEC, d.NutrientPID.State.ControlSignal, wantNutrient, d.WantNutrient)
	if wantNutrient > d.WantNutrient {
		d.WantNutrient = wantNutrient
	}
	d.streamStatusUpdate()
}
