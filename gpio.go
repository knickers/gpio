package gpio

import (
	"errors"
	"fmt"
	"os"
)

type Direction int
type State int

const (
	OUTPUT = Direction(0)
	INPUT  = Direction(1)
	HIGH   = State(1)
	LOW    = State(0)
)

func (d *Direction) String() string {
	if *d == INPUT {
		return "in"
	}
	return "out"
}

func (d *State) String() string {
	if *d == HIGH {
		return "high"
	}
	return "low"
}

func (d *State) NumString() string {
	if *d == HIGH {
		return "1"
	}
	return "0"
}

type Pin struct {
	fd    *os.File
	num   uint
	dir   Direction
	state State
}

func (p *Pin) enable_export() error {
	_, err := os.Stat(fmt.Sprintf("/sys/class/gpio/gpio%d", p.num))
	if err == nil {
		// already exported
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		// some other error
		return err
	}
	fd, err := os.OpenFile("/sys/class/gpio/export", os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fmt.Fprintf(fd, "%d\n", p.num)
	return err
}

func (p *Pin) SetDirection(dir Direction) error {
	if p.dir != dir {
		fd, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/direction", p.num), os.O_WRONLY|os.O_SYNC, 0666)
		if err != nil {
			return err
		}
		defer fd.Close()
		fmt.Fprintln(fd, dir.String())
		p.dir = dir
	}
	return nil
}

func (p *Pin) GetDirection() Direction {
	return p.dir
}

func (p *Pin) SetState(st State) error {
	var err error
	if p.state != st {
		if p.dir != OUTPUT {
			return errors.New("This pin is set to " + p.dir.String())
		}
		fmt.Println("Setting pin", p.num, "to", st.String())
		_, err = fmt.Fprintln(p.fd, st.NumString())
	}
	return err
}

func (p *Pin) GetState() (st State, err error) {
	if p.dir == INPUT {
		st, err = p.state, nil
	} else if p.dir == OUTPUT {
		_, err = fmt.Fscan(p.fd, st)
	}
	return
}

func (p *Pin) GetNumber() uint {
	return p.num
}

func (p *Pin) Close() {
	p.fd.Close()
}

func NewPin(num uint, dir Direction) (gpio *Pin, err error) {
	gpio = new(Pin)
	gpio.num = num

	if err = gpio.enable_export(); err != nil {
		return nil, err
	}
	if err = gpio.SetDirection(dir); err != nil {
		return nil, err
	}
	if gpio.fd, err = os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/value", gpio.num), os.O_WRONLY|os.O_SYNC, 0666); err != nil {
		return nil, err
	}
	return gpio, nil
}
