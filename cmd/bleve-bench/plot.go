package main

import "github.com/gonum/plot"
import "github.com/gonum/plot/plotter"
import "github.com/gonum/plot/plotutil"
import "github.com/gonum/plot/vg"
import "fmt"
import "strings"

type Line struct {
	Pt                 plotter.XYs
	ConfName, TypeName string
}

func NewLines(n int, o int, conf string, name []string) []*Line {
	l := make([]*Line, o)
	sn := strings.Split(conf, "/")
	for i := 0; i < o; i++ {
		l[i] = &Line{
			Pt:       make(plotter.XYs, n),
			ConfName: sn[len(sn)-1],
		}
	}
	return l
}

func doPlot(l []*Line, name string, xname string, yname string, file string) {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = name
	p.X.Label.Text = xname
	p.Y.Label.Text = yname

	for i, k := range l {
		err = AddLinePointsWithColor(p, i, k.ConfName, k.Pt)
		if err != nil {
			panic(err)
		}
	}
	// Save the plot to a PNG file.
	if err := p.Save(4*vg.Inch, 4*vg.Inch, file); err != nil {
		panic(err)
	}
}

func AddLinePointsWithColor(plt *plot.Plot, i int, vs ...interface{}) error {
	var ps []plot.Plotter
	names := make(map[[2]plot.Thumbnailer]string)
	name := ""
	for _, v := range vs {
		switch t := v.(type) {
		case string:
			name = t

		case plotter.XYer:
			l, s, err := plotter.NewLinePoints(t)
			if err != nil {
				return err
			}
			l.Color = plotutil.Color(i)
			l.Dashes = plotutil.Dashes(i)
			s.Color = plotutil.Color(i)
			s.Shape = plotutil.Shape(i)
			ps = append(ps, l, s)
			if name != "" {
				names[[2]plot.Thumbnailer{l, s}] = name
				name = ""
			}

		default:
			panic(fmt.Sprintf("AddLinePointsWithColor handles strings and plotter.XYers, got %T", t))
		}
	}
	plt.Add(ps...)
	for ps, n := range names {
		plt.Legend.Add(n, ps[0], ps[1])
	}
	return nil
}
