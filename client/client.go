package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"

	pb "ds/polyrpc/dspoly"
)

var (
	serverAddr = flag.String("s", "localhost:6090", "grpc server address")
	timeout = flag.Duration("t", 5 * time.Second, "grpc dial timeout")
)

const (
	AnsiClear = "\033[H\033[2J"
)

func formatPolynomial(p *pb.Polynomial) string {
	cs := make(map[int]float64)
	for i, c := range p.Coefficients {
		if c != 0 {
			cs[i] = c
		}
	}

	if len(cs) == 0 {
		return "0"
	}

	repr := ""
	for i := len(p.Coefficients) - 1; i >= 0; i-- {
		c, ok := cs[i]
		if !ok {
			continue
		}

		var cstr string
		if len(repr) > 0 {
			if c > 0.0 {
				repr += " + "
			} else {
				repr += " - "
			}

			if i == 0 || (c != 1.0 && c != -1.0) {
				cstr = fmt.Sprintf("%g", math.Abs(c))
			}
		} else {
			switch {
			case i == 0 || (c != 1.0 && c != -1.0):
				cstr = fmt.Sprintf("%g", c)
			case c == -1.0:
				cstr = "-"
			}
		}

		switch {
		case i == 0:
			repr += cstr
		case i == 1:
			repr += fmt.Sprintf("%sx", cstr)
		default:
			repr += fmt.Sprintf("%sx^%d", cstr, i)
		}
	}

	return repr
}

type coefficientError string

func (e coefficientError) Error() string {
	return fmt.Sprintf("bad coefficient: %s", string(e))
}

func readPolynomial(reader *bufio.Reader) (p *pb.Polynomial, err error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	line = strings.Trim(line, " \t\r\n")
	words := regexp.MustCompile(`\s+`).Split(line, -1)
	coeffs := make([]float64, len(words))

	for i := range words {
		var c float64
		w := &words[len(words) - 1 - i]
		c, err = strconv.ParseFloat(*w, 64)

		if err != nil {
			err = coefficientError(*w)
			return
		}

		coeffs[i] = c
	}

	p = &pb.Polynomial{Coefficients: coeffs}
	return
}

type menuOption struct {
	key string
	text string
	action func() error;
}

func clientApp(client pb.DSPolyClient) {
	reader := bufio.NewReader(os.Stdin)
	p := &pb.Polynomial{};
	q := &pb.Polynomial{};

	ret2con := func() {
		fmt.Fprint(os.Stderr, "press return to continue\n")
		reader.ReadString('\n')
	}

	polyop := func(opstr string, op func(context.Context, *pb.PolynomialTuple, ...grpc.CallOption) (*pb.Polynomial, error)) (err error) {
		polys := &pb.PolynomialTuple{Polys: []*pb.Polynomial{p, q}}
		res, err := op(context.Background(), polys)

		if err == nil {
			fmt.Printf("p %s q = [%s]\n", opstr, formatPolynomial(res))
			ret2con()
		}

		return
	}

	printError := func(what string, err error) {
		log.Printf("failed to %s: %s\n", what, err)
		ret2con()
	}

	setp := func() (err error) {
		fmt.Print("p? ")
		rp, err := readPolynomial(reader)
		fmt.Println()

		if rp != nil {
			p = rp
		}

		return
	}

	setq := func() (err error) {
		fmt.Print("q? ")
		rp, err := readPolynomial(reader)
		fmt.Println()

		if rp != nil {
			q = rp
		}

		return
	}

	add := func() error {
		return polyop("+", client.Add)
	}

	sub := func() error {
		return polyop("-", client.Sub)
	}

	mul := func() error {
		return polyop("*", client.Mul)
	}

	menuOpts := []*menuOption{
		{"1", "set p", setp},
		{"2", "set q", setq},
		{"3", "Add(p, q)", add},
		{"4", "Sub(p, q)", sub},
		{"5", "Mul(p, q)", mul},
	}

MenuLoop:
	for {
		fmt.Print(AnsiClear)
		fmt.Printf("\np = [%s]\nq = [%s]\n\n", formatPolynomial(p), formatPolynomial(q))
		for _, o := range menuOpts {
			fmt.Printf("%s) %s\n", o.key, o.text)
		}

	InputLoop:
		for {
			fmt.Print("#? ")
			line, err := reader.ReadString('\n')
			fmt.Println()

			if err != nil {
				if err == io.EOF {
					break MenuLoop
				}

				printError("read", err)
				break
			}

			if len(line) == 0 {
				continue
			}

			line = strings.Trim(line, " \t\r\n")
			for _, o := range menuOpts {
				if line == o.key {
					err = o.action()
					if err != nil {
						if err == io.EOF {
							break MenuLoop
						}

						printError(o.text, err)
					}

					break InputLoop
				}
			}
		}
	}
}

func main() {
	flag.Parse()
	conn, err := grpc.Dial(*serverAddr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(*timeout))

	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}

	defer conn.Close()
	clientApp(pb.NewDSPolyClient(conn))
}
