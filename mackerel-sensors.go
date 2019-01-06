package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type It8686 struct {
	CPU_Vcore struct {
		Input float64 `json:"in0_input"`
	} `json:"CPU Vcore"`
	DRAM_V struct {
		Input float64 `json:"in6_input"`
	} `json:"DRAM Channel A/B"`
	CPU_Fan struct {
		Input float64 `json:"fan1_input"`
	} `json:"CPU_FAN"`
	SYS_Fan1 struct {
		Input float64 `json:"fan2_input"`
	} `json:"SYS_FAN1"`
	SYS_Fan2 struct {
		Input float64 `json:"fan3_input"`
	} `json:"SYS_FAN2"`
	Chipset_Temp struct {
		Input float64 `json:"temp2_input"`
	} `json:"Chipset Temp"`
	CPU_Temp struct {
		Input float64 `json:"temp3_input"`
	} `json:"CPU Temp"`
	PCIEX16_Temp struct {
		Input float64 `json:"temp4_input"`
	} `json:"PCI-EX16 Temp"`
	VRMMOS_Temp struct {
		Input float64 `json:"temp5_input"`
	} `json:"VRM MOS Temp"`
	VSOCMOS_Temp struct {
		Input float64 `json:"temp6_input"`
	} `json:"vSOC MOS Temp"`
}

type Sensors struct {
	Sensor It8686 `json:"it8686-isa-0a40"`
}

type Metric struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

type Graph struct {
	Label   string   `json:"label"`
	Unit    string   `json:"unit"`
	Metrics []Metric `json:"metrics"`
}

type Graphs struct {
	Fan  Graph `json:"sensors.fan"`
	Temp Graph `json:"sensors.temp"`
}

type Meta struct {
	Graphs Graphs `json:"graphs"`
}

func main() {
	var err error
	if os.Getenv("MACKEREL_AGENT_PLUGIN_META") == "1" {
		err = graph()
	} else {
		err = sensor()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func graph() error {
	fmt.Fprintf(os.Stdout, "# mackerel-agent-plugin\n")

	out := Meta{
		Graphs: Graphs{
			Fan: Graph{
				Label: "Temp",
				Unit:  "integer",
				Metrics: []Metric{
					Metric{
						Name:  "cpu",
						Label: "CPU_FAN",
					},
					Metric{
						Name:  "sys1",
						Label: "SYS_FAN1",
					},
					Metric{
						Name:  "sys2",
						Label: "SYS_FAN1",
					},
				},
			},
			Temp: Graph{
				Label: "Fan",
				Unit:  "integer",
				Metrics: []Metric{
					Metric{
						Name:  "chipset",
						Label: "Chipset Temp",
					},
					Metric{
						Name:  "cpu",
						Label: "CPU Temp",
					},
					Metric{
						Name:  "pciex16",
						Label: "PCI-EX16 Temp",
					},
					Metric{
						Name:  "vrm",
						Label: "VRM MOS Temp",
					},
					Metric{
						Name:  "vsoc",
						Label: "vSOC MOS Temp",
					},
				},
			},
		},
	}
	return json.NewEncoder(os.Stdout).Encode(&out)
}

func sensor() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cmdbuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "sensors", "-j")
	cmd.Stdout = &cmdbuf
	//cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	var ss Sensors
	derr := json.NewDecoder(&cmdbuf).Decode(&ss)
	if derr != nil {
		return derr
	}
	ut := time.Now().Unix()
	fmt.Fprintf(os.Stdout, "sensors.fan.cpu\t%d\t%d\n", int64(ss.Sensor.CPU_Fan.Input), ut)
	fmt.Fprintf(os.Stdout, "sensors.fan.sys1\t%d\t%d\n", int64(ss.Sensor.SYS_Fan1.Input), ut)
	fmt.Fprintf(os.Stdout, "sensors.fan.sys2\t%d\t%d\n", int64(ss.Sensor.SYS_Fan2.Input), ut)

	fmt.Fprintf(os.Stdout, "sensors.temp.chipset\t%d\t%d\n", int64(ss.Sensor.Chipset_Temp.Input), ut)
	fmt.Fprintf(os.Stdout, "sensors.temp.cpu\t%d\t%d\n", int64(ss.Sensor.CPU_Temp.Input), ut)
	fmt.Fprintf(os.Stdout, "sensors.temp.pciex16\t%d\t%d\n", int64(ss.Sensor.PCIEX16_Temp.Input), ut)
	fmt.Fprintf(os.Stdout, "sensors.temp.vrm\t%d\t%d\n", int64(ss.Sensor.VRMMOS_Temp.Input), ut)
	fmt.Fprintf(os.Stdout, "sensors.temp.vsoc\t%d\t%d\n", int64(ss.Sensor.VSOCMOS_Temp.Input), ut)
	return nil
}
