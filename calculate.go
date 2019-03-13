package main

import (
	"fmt"
	"log"
	"math"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func main() {
	var mw *walk.MainWindow
	var db *walk.DataBinder
	pmvInput := new(PMVInput)
	var acceptPB *walk.PushButton
	MainWindow{
		AssignTo: &mw,
		Icon:     "2",
		Title:    "计算PMV的小程序",
		Layout:   VBox{},
		MinSize:  Size{400, 200},
		DataBinder: DataBinder{
			AssignTo:       &db,
			DataSource:     pmvInput,
			ErrorPresenter: ToolTipErrorPresenter{},
		},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{
						Text: "空气温度:",
					},
					NumberEdit{
						Value:    Bind("Ta", Range{-100.00, 100.00}),
						Suffix:   " °C",
						Decimals: 2,
					},
					Label{
						Text: "黑球温度",
					},
					NumberEdit{
						Value:    Bind("Tr", Range{-100.00, 100.00}),
						Suffix:   " °C",
						Decimals: 2,
					},
					Label{
						Text: "相对湿度",
					},
					NumberEdit{
						Value:    Bind("Rh", Range{0.00, 100.00}),
						Suffix:   " %",
						Decimals: 2,
					},
					Label{
						Text: "空气流速",
					},
					NumberEdit{
						Value:    Bind("Vel", Range{0.00, 100.00}),
						Suffix:   " m/s",
						Decimals: 2,
					},
					Label{
						Text: "人体代谢率",
					},
					NumberEdit{
						Value:    Bind("Met", Range{0.00, 100.00}),
						Suffix:   " met",
						Decimals: 2,
					},
					Label{
						Text: "服装热阻:",
					},
					NumberEdit{
						Value:    Bind("Clo", Range{0.00, 99.99}),
						Suffix:   " clo",
						Decimals: 2,
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "OK",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								log.Print(err)
								return
							}
							PMVvalue, PPDvalue := pmv(pmvInput.Clo, pmvInput.Ta, pmvInput.Tr, pmvInput.Met, pmvInput.Vel, pmvInput.Rh)
							walk.MsgBox(mw, "结果", fmt.Sprintf("预测平均评价(PMV)为：%.2f\r\n预测不满意百分数(PPD)：%.2f",
								PMVvalue, PPDvalue), walk.MsgBoxOK)
						},
					},
				},
			},
		},
	}.Run()
}

type PMVInput struct {
	Clo, Ta, Tr, Met, Vel, Rh float64
	// Clo, Ta float64
}

func pmv(clo, ta, tr, met, vel, rh float64) (PMV, PPD float64) {
	// 计算水蒸气分压力
	FNPS := math.Exp(16.6536 - 4030.183/(ta+235))
	PA := rh * 10 * FNPS

	// 转换服装热阻和代谢率单位
	ICL := 0.156 * clo
	M := met * 58.15

	// 通过服装热阻计算fcl服装表面积比
	var FCL float64 = 1
	if ICL < 0.078 {
		FCL = 1 + 1.29*ICL
	} else {
		FCL = 1.05 + 0.645*ICL
	}

	//计算对流系数
	HCF := 12.1 * math.Pow(vel, 0.5)

	//转开示温度
	TAA := ta + 273
	TRA := tr + 273

	//衣服外表面空气开式温度？
	TCLA := TAA + (35.5-ta)/(3.5*(6.45*ICL+0.1))

	P1 := ICL * FCL
	P2 := P1 * 3.96
	P3 := P1 * 100
	P4 := P1 * TAA
	P5 := 308.7 - 0.028*M + P2*math.Pow(TRA/100, 4)
	XN := TCLA / 100
	XF := TCLA / 50
	N := 0
	EPS := 0.00015

	//数值计算得到XN，最终目的是为了得到TCL
	var HCN, HC float64
	for math.Abs(XN-XF) > EPS {
		XF = (XF + XN) / 2
		HCF = 12.1 * math.Pow(vel, 0.5)
		HCN = 2.38 * math.Pow(math.Abs(100*XF-TAA), 0.25)

		if HCF > HCN {
			HC = HCF
		} else {
			HC = HCN
		}
		XN = (P5 + P4*HC - P2*math.Pow(XF, 4)) / (100 + P3*HC)
		N++
	}
	TCL := 100*XN - 273

	//皮肤扩散蒸发损失
	HL1 := 3.05 * 0.001 * (5733 - 6.99*M - PA)

	//sweat loss
	var HL2 float64
	if M > 58.15 {
		HL2 = 0.42 * (M - 58.15)
	}

	//Latent respiration loss
	HL3 := 1.7 * 0.00001 * M * (5867 - PA)

	//Dry respiration loss
	HL4 := 0.0014 * M * (34 - ta)

	//Radiation loss
	HL5 := 3.96 * FCL * (math.Pow(XN, 4) - math.Pow((TRA/100), 4))

	HL6 := FCL * HC * (TCL - ta)

	// Thermal sensation to skin tran coef
	TS := 0.303*math.Exp(-0.036*M) + 0.028

	PMV = TS * (M - HL1 - HL2 - HL3 - HL4 - HL5 - HL6)
	PPD = 100 - 95*math.Exp(-0.03353*math.Pow(PMV, 4)-0.2179*math.Pow(PMV, 2))
	return
}
