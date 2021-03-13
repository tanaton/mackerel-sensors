package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Nct6798 struct {
	Adapter string
	Fan1    struct {
		Input float64 `json:"fan1_input"`
	} `json:"fan1"`
	Fan2 struct {
		Input float64 `json:"fan2_input"`
	} `json:"fan2"`
	Fan3 struct {
		Input float64 `json:"fan3_input"`
	} `json:"fan3"`
	Fan4 struct {
		Input float64 `json:"fan4_input"`
	} `json:"fan4"`
	Fan5 struct {
		Input float64 `json:"fan5_input"`
	} `json:"fan5"`
	Fan6 struct {
		Input float64 `json:"fan6_input"`
	} `json:"fan6"`
	Fan7 struct {
		Input float64 `json:"fan7_input"`
	} `json:"fan7"`
	Systin struct {
		Input float64 `json:"temp1_input"`
	} `json:"SYSTIN"`
	Cputin struct {
		Input float64 `json:"temp2_input"`
	} `json:"CPUTIN"`
	Auxtin0 struct {
		Input float64 `json:"temp3_input"`
	} `json:"AUXTIN0"`
	Auxtin1 struct {
		Input float64 `json:"temp4_input"`
	} `json:"AUXTIN1"`
	Auxtin2 struct {
		Input float64 `json:"temp5_input"`
	} `json:"AUXTIN2"`
	Auxtin3 struct {
		Input float64 `json:"temp6_input"`
	} `json:"AUXTIN3"`
	PeciAgent0Calibration struct {
		Input float64 `json:"temp7_input"`
	} `json:"PECI Agent 0 Calibration"`
	PchChipCPUMaxTemp struct {
		Input float64 `json:"temp8_input"`
	} `json:"PCH_CHIP_CPU_MAX_TEMP"`
	PchChipTemp struct {
		Input float64 `json:"temp9_input"`
	} `json:"PCH_CHIP_TEMP"`
	PchCPUTemp struct {
		Input float64 `json:"temp10_input"`
	} `json:"PCH_CPU_TEMP"`
}

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

type CPUTemp struct {
	Tctl struct {
		Input float64 `json:"temp1_input"`
	}
	Tdie struct {
		Input float64 `json:"temp2_input"`
	}
	Tccd1 struct {
		Input float64 `json:"temp3_input"`
	}
}

type Sensors struct {
	Nct6798 *Nct6798 `json:"nct6798-isa-0290"`
	It8686  *It8686  `json:"it8686-isa-0a40"`
	K10Temp *CPUTemp `json:"k10temp-pci-00c3"`
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
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		os.Interrupt,
		os.Kill,
	)
	defer stop()
	err := _main(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func _main(ctx context.Context) error {
	var err error
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var cmdbuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "sensors", "-j")
	cmd.Stdout = &cmdbuf
	//cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	var ss Sensors
	err = json.NewDecoder(&cmdbuf).Decode(&ss)
	if err != nil {
		return err
	}
	if os.Getenv("MACKEREL_AGENT_PLUGIN_META") == "1" {
		err = graph(ctx, &ss)
	} else {
		err = sensor(ctx, &ss)
	}
	return err
}

func graph(ctx context.Context, ss *Sensors) error {
	fmt.Fprintf(os.Stdout, "# mackerel-agent-plugin\n")

	out := Meta{
		Graphs: Graphs{
			Fan: Graph{
				Label:   "Fan",
				Unit:    "integer",
				Metrics: []Metric{},
			},
			Temp: Graph{
				Label: "Temp",
				Unit:  "integer",
				Metrics: []Metric{
					{
						Name:  "air",
						Label: "Air Temp",
					},
				},
			},
		},
	}

	if ss.It8686 != nil {
		out.Graphs.Fan.Metrics = append(out.Graphs.Fan.Metrics, []Metric{
			{
				Name:  "cpu",
				Label: "CPU_FAN",
			},
			{
				Name:  "sys1",
				Label: "SYS_FAN1",
			},
			{
				Name:  "sys2",
				Label: "SYS_FAN2",
			},
		}...)

		out.Graphs.Temp.Metrics = append(out.Graphs.Temp.Metrics, []Metric{
			{
				Name:  "chipset",
				Label: "Chipset Temp",
			},
			{
				Name:  "cpu",
				Label: "CPU Temp",
			},
			{
				Name:  "pciex16",
				Label: "PCI-EX16 Temp",
			},
			{
				Name:  "vrm",
				Label: "VRM MOS Temp",
			},
			{
				Name:  "vsoc",
				Label: "vSOC MOS Temp",
			},
		}...)
	}
	if ss.Nct6798 != nil {
		out.Graphs.Fan.Metrics = append(out.Graphs.Fan.Metrics, []Metric{
			{
				Name:  "cpu",
				Label: "CPU_FAN",
			},
			{
				Name:  "cpuopt",
				Label: "CPU_OPT_FAN",
			},
			{
				Name:  "sys1",
				Label: "SYS_FAN1",
			},
			{
				Name:  "sys2",
				Label: "SYS_FAN2",
			},
			{
				Name:  "sys3",
				Label: "SYS_FAN3",
			},
		}...)

		out.Graphs.Temp.Metrics = append(out.Graphs.Temp.Metrics, []Metric{
			{
				Name:  "motherboard",
				Label: "Motherboard Temp",
			},
			{
				Name:  "cpu",
				Label: "CPU Temp",
			},
		}...)
	}
	if ss.K10Temp != nil {
		out.Graphs.Temp.Metrics = append(out.Graphs.Temp.Metrics, []Metric{
			{
				Name:  "tctl",
				Label: "Tctl UEFI CPU Temp",
			},
			{
				Name:  "tdie",
				Label: "Tdie CPU Temp",
			},
			{
				Name:  "tccd1",
				Label: "Tccd1 CPU Temp",
			},
		}...)
	}
	return json.NewEncoder(os.Stdout).Encode(&out)
}

func sensor(ctx context.Context, ss *Sensors) error {
	ut := time.Now().Unix()
	if ss.It8686 != nil {
		fmt.Fprintf(os.Stdout, "sensors.fan.cpu\t%d\t%d\n", int64(ss.It8686.CPU_Fan.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.fan.sys1\t%d\t%d\n", int64(ss.It8686.SYS_Fan1.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.fan.sys2\t%d\t%d\n", int64(ss.It8686.SYS_Fan2.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.temp.chipset\t%d\t%d\n", int64(ss.It8686.Chipset_Temp.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.temp.cpu\t%d\t%d\n", int64(ss.It8686.CPU_Temp.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.temp.pciex16\t%d\t%d\n", int64(ss.It8686.PCIEX16_Temp.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.temp.vrm\t%d\t%d\n", int64(ss.It8686.VRMMOS_Temp.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.temp.vsoc\t%d\t%d\n", int64(ss.It8686.VSOCMOS_Temp.Input), ut)
	}
	if ss.Nct6798 != nil {
		fmt.Fprintf(os.Stdout, "sensors.fan.cpu\t%d\t%d\n", int64(ss.Nct6798.Fan2.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.fan.cpuopt\t%d\t%d\n", int64(ss.Nct6798.Fan7.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.fan.sys1\t%d\t%d\n", int64(ss.Nct6798.Fan1.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.fan.sys2\t%d\t%d\n", int64(ss.Nct6798.Fan3.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.fan.sys3\t%d\t%d\n", int64(ss.Nct6798.Fan4.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.temp.motherboard\t%d\t%d\n", int64(ss.Nct6798.Systin.Input), ut)
		fmt.Fprintf(os.Stdout, "sensors.temp.cpu\t%d\t%d\n", int64(ss.Nct6798.Cputin.Input), ut)
	}
	if ss.K10Temp != nil {
		fmt.Fprintf(os.Stdout, "sensors.temp.tctl\t%.3f\t%d\n", ss.K10Temp.Tctl.Input, ut)
		fmt.Fprintf(os.Stdout, "sensors.temp.tdie\t%.3f\t%d\n", ss.K10Temp.Tdie.Input, ut)
		fmt.Fprintf(os.Stdout, "sensors.temp.tccd1\t%.3f\t%d\n", ss.K10Temp.Tccd1.Input, ut)
	}
	return air(ctx, ut)
}

func air(ctx context.Context, ut int64) error {
	// rootで呼ばないとダメ？
	var cmdbuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "/home/tanaton/src/TEMPered/utils/tempered")
	cmd.Stdout = &cmdbuf
	//cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(&cmdbuf)
	if scanner.Scan() {
		list := strings.Split(scanner.Text(), " ")
		if len(list) > 3 {
			s, err := strconv.ParseFloat(list[3], 64)
			if err == nil {
				fmt.Fprintf(os.Stdout, "sensors.temp.air\t%.2f\t%d\n", s, ut)
				return nil
			}
		}
	}
	fmt.Fprintf(os.Stdout, "sensors.temp.air\t%d\t%d\n", 20, ut)
	return errors.New("気温の取得に失敗")
}
