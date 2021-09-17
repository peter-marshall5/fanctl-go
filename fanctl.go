package main

import (
  "fmt"
  "os"
  "time"
  "math"
  "strconv"
  "strings"
  "os/signal"
  "syscall"
  "io/ioutil"
  "flag"
)

const (
  minFanSpeed = float64(28)
  medFanSpeed = float64(35)
  highTemp = float64(70)
  critTemp = float64(85)
  dangerousTemp = float64(94)
  sustainedLoadTime = 14
  tempThreshold = float64(4)
  normalSpeedChangeRate = float64(1)
  critModeSpeedChangeRate = float64(4)
  highModeReductionRate = float64(1)
)

var speedSatisfied = false
var speedTarget = float64(10)
var currSpeed = float64(0)
var mode = 0
var currTemp = float64(20)
var oldTemp = float64(0)
var highTempTimer = int64(0)
var speedChangeRate = float64(4)
var lastECVal = 0

var gracefulQuitTried = false

var thermalZone string
var ecPath string
var ecAddr int64
var manualAddr int64
var readAddr int64
var ecMin float64
var ecMax float64
var readMin int64
var readMax int64
var debug bool

func quit(msg error) {
  if (!gracefulQuitTried) {
    gracefulQuitTried = true
    disableManualControl()
  }
  panic(msg)
}

func main() {
  flag.StringVar(&thermalZone, "thermal-zone", "/sys/class/hwmon/hwmon5/temp2_input", "Path to CPU temperature reading")
  flag.StringVar(&ecPath, "ec-path", "/dev/ec", "Path to embedded controller interface")
  flag.Int64Var(&ecAddr, "ec-addr", 25, "Address for fan speed control register")
  flag.Int64Var(&manualAddr, "manual-addr", 21, "Address for manual control enable register")
  flag.Int64Var(&readAddr, "read-addr", 17, "Address for current speed register")
  flag.Float64Var(&ecMin, "ec-min", 0, "Minimum value to write to speed control register")
  flag.Float64Var(&ecMax, "ec-max", 48, "Maximum value to write to speed control register")
  flag.Int64Var(&readMin, "read-min", 14, "Minimum value that can be read from current speed register")
  flag.Int64Var(&readMax, "read-max", 54, "Maximum value that can be read from current speed register")
  flag.BoolVar(&debug, "debug", false, "Debug output")
  flag.Parse()

  fmt.Println("Fan control script by petmshall")
  if debug {
    fmt.Println("Debug output enabled")
  }
  setupCloseHandler()
  loop()
  ticks := 0
  for range time.Tick(200 * time.Millisecond) {
    ticks++
    if ticks > 5 {
      loop()
      ticks = 0
    }
    speedLoop()
  }
}

func loop() {
  checkManualControl()
  tempLoop()
}

func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\rHanding fan control over to BIOS")
    disableManualControl()
		os.Exit(0)
	}()
}

func checkManualControl() {
  if readEC(manualAddr) == 0 {
    if debug {
      fmt.Println("Activating manual control")
    }
    speedTarget = 0
    mode = 0
    enableManualControl()
    writeSpeed()
  }
}

func enableManualControl() {
  writeEC(manualAddr, 1)
}

func disableManualControl() {
  writeEC(manualAddr, 0)
}

func writeEC(address int64, value int) {
  f, err := os.OpenFile(ecPath, os.O_RDWR, 0644)
  if err != nil {
    quit(err)
  }
  defer f.Close()
  if _, err := f.WriteAt([]byte{byte(value)}, address); err != nil {
    quit(err)
  }
}

func readEC(address int64) int {
  f, err := os.OpenFile(ecPath, os.O_RDWR, 0644)
  if err != nil {
    quit(err)
  }
  defer f.Close()
  b := make([]byte, 1)
  f.ReadAt(b, address)
  return int(b[0])
}

func readTemp() float64 {
  f, err := ioutil.ReadFile(thermalZone)
  if err != nil {
    quit(err)
  }
  tempInt, _ := strconv.Atoi(strings.TrimRight(string(f), "\n"))
  return float64(tempInt) / 1000
}

func tempLoop() {
  currTemp = readTemp()
  if currTemp >= dangerousTemp {
    fmt.Println("Dangerous CPU temperature, maxing out fan")
    currSpeed = 100
    writeSpeed()
    return
  }
  currTime := time.Now().Unix()
  if highTempTimer > 0 {
    if currTemp < highTemp {
      if debug {
        fmt.Println("Cancelling timer")
      }
      highTempTimer = 0
    } else {
      timeDelta := currTime - highTempTimer
      if timeDelta > sustainedLoadTime {
        if debug {
          fmt.Println("Switching to high temperature mode")
        }
        mode = 1
        highTempTimer = 0
        updateSpeed()
      }
    }
  }
  tempDelta := currTemp - oldTemp
  if mode == 0 {
    if currTemp > highTemp && highTempTimer == 0 {
      if debug {
        fmt.Println("High temperature, starting timer")
      }
      highTempTimer = currTime
      speedTarget = medFanSpeed
      return
    }
    if math.Abs(tempDelta) > tempThreshold {
      oldTemp = currTemp
      updateSpeed()
    }
  } else if mode == 1 {
    if currTemp < highTemp - tempThreshold {
      if debug {
        fmt.Println("Switching to low temperature mode")
      }
      mode = 0
      //updateSpeed()
    } else if currTemp > critTemp {
      if debug {
        fmt.Println("Switching to critical mode")
      }
      mode = 2
    } else if math.Abs(tempDelta) > tempThreshold {
      oldTemp = currTemp
      updateSpeed()
    }
  } else {
    if currTemp < critTemp - tempThreshold {
      if debug {
        fmt.Println("Switching to high temperature mode")
      }
      mode = 1
    } else {
      updateSpeed()
    }
  }
}

func speedLoop() {
  if mode == 2 {
    speedChangeRate = critModeSpeedChangeRate
  } else {
    speedChangeRate = normalSpeedChangeRate
  }
  if math.Abs(speedTarget - currSpeed) > speedChangeRate {
    if speedTarget > currSpeed {
      currSpeed += math.Min(speedTarget - currSpeed, speedChangeRate)
    } else {
      if (mode == 0) {
        currSpeed -= math.Min(currSpeed - speedTarget, speedChangeRate)
      } else {
        currSpeed -= math.Min(currSpeed - speedTarget, highModeReductionRate)
      }
    }
    writeSpeed()
  } else if !speedSatisfied {
    if debug {
      fmt.Println("Speed satisfied")
    }
    currSpeed = speedTarget
    speedSatisfied = true
    writeSpeed()
  }
}

func writeSpeed() {
  ecVal := int(math.Floor(currSpeed / 100 * (ecMax - ecMin) + ecMin))
  if ecVal != lastECVal {
    lastECVal = ecVal
    writeEC(ecAddr, ecVal)
    if debug {
      fmt.Print("Speed: ", currSpeed)
      fmt.Println("%, Wrote EC value:", ecVal)
    }
  }
}

func updateSpeed() {
  if mode == 1 {
    speedTarget = calculateHighSpeed()
  } else if mode == 0 {
    speedTarget = calculateLowSpeed()
    if (speedTarget <= minFanSpeed) {
      speedTarget = 0
    }
  } else if mode == 2 {
    speedTarget = calculateCritSpeed()
  }
}

func calculateLowSpeed() float64 {
  return math.Min(math.Round(math.Pow(currTemp, 2) * currTemp / 11842), medFanSpeed)
}

func calculateHighSpeed() float64 {
  return math.Min(math.Max(math.Round(currTemp * 2.7 - 157), medFanSpeed), 100)
}

func calculateCritSpeed() float64 {
  return math.Min(math.Max(math.Round(currTemp * 2.9 - 172), medFanSpeed), 100)
}
