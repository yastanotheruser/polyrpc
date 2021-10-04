package main

import (
	"context"
	"flag"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"
	pb "ds/polyrpc/dspoly"
)

var (
	port = flag.Uint("p", 6090, "server tcp port")
)

type dsPolyServer struct {
	pb.UnimplementedDSPolyServer
}

func polynomialDegree(p *pb.Polynomial) int {
	l := len(p.Coefficients)
	if l > 0 {
		return l - 1
	} else {
		return 0
	}
}

func makeSumPolynomial(ps *pb.PolynomialTuple) *pb.Polynomial {
	n := 0
	for _, p := range ps.Polys {
		if len(p.Coefficients) > n {
			n = len(p.Coefficients)
		}
	}

	return &pb.Polynomial{Coefficients: make([]float64, n)}
}

func makeProductPolynomial(ps *pb.PolynomialTuple) *pb.Polynomial {
	n := 1
	for _, p := range ps.Polys {
		if isZeroPolynomial(p) {
			n = 0
			break
		}

		n += polynomialDegree(p)
	}

	coeffs := make([]float64, n)
	if n > 0 {
		copy(coeffs, ps.Polys[0].Coefficients)
	}

	return &pb.Polynomial{Coefficients: coeffs}
}

func isZeroPolynomial(p *pb.Polynomial) bool {
	for _, c := range p.Coefficients {
		if c != 0.0 {
			return false
		}
	}

	return true
}

func (s *dsPolyServer) Add(ctx context.Context, ps *pb.PolynomialTuple) (*pb.Polynomial, error) {
	res := makeSumPolynomial(ps)
	for _, p := range ps.Polys {
		for i, c := range p.Coefficients {
			res.Coefficients[i] += c
		}
	}

	return res, nil
}

func (s *dsPolyServer) Sub(ctx context.Context, ps *pb.PolynomialTuple) (*pb.Polynomial, error) {
	res := makeSumPolynomial(ps)
	if len(ps.Polys) >= 1 {
		copy(res.Coefficients, ps.Polys[0].Coefficients)
	}

	for _, p := range ps.Polys[1:] {
		for i, c := range p.Coefficients {
			res.Coefficients[i] -= c
		}
	}

	return res, nil
}

func (s *dsPolyServer) Mul(ctx context.Context, ps *pb.PolynomialTuple) (*pb.Polynomial, error) {
	res := makeProductPolynomial(ps)
	if len(res.Coefficients) == 0 {
		return res, nil
	}

	deg := polynomialDegree(ps.Polys[0])
	for _, p := range ps.Polys[1:] {
		coeffs := make([]float64, len(res.Coefficients))
		for i := 0; i <= deg; i++ {
			c := res.Coefficients[i]
			for j, d := range p.Coefficients {
				coeffs[i + j] += c * d
			}
		}

		copy(res.Coefficients, coeffs)
		deg += polynomialDegree(p)
	}

	return res, nil
}

func newServer() *dsPolyServer {
	s := &dsPolyServer{}
	return s
}

func main() {
	flag.Parse()

	ln, err := net.Listen("tcp", ":" + strconv.FormatUint(uint64(*port), 10))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDSPolyServer(grpcServer, newServer())
	grpcServer.Serve(ln)
}
