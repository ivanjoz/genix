import bb, { spline } from "billboard.js";
import * as d3 from "d3";
import { createEffect } from 'solid-js';
import { PageContainer } from '~/core/page';

// 2) import css if your dev-env supports. If don't, include them via <link>
import "billboard.js/dist/billboard.css";

// or theme style. Find more themes from 'theme' folder
import "billboard.js/dist/theme/insight.css";
import { DatePicker } from '~/components/Datepicker';
import { ImageUploader } from '~/components/Uploaders';

export default function ChartTest() {
  
  createEffect(() => {
    var chart = bb.generate({
      data: {
        xs: {
          data1: "x1",
          data2: "x2"
        },
        columns: [
          ["x1", 10, 30, 45, 50, 70, 100],
          ["x2", 30, 50, 75, 100, 120],
          ["data1", 30, 200, 100, 400, 150, 250],
          ["data2", 20, 180, 240, 100, 190]
        ],
        type: spline(), // for ESM specify as: line()
      },
      padding: {
        top: 40,
        right: 100,
        bottom: 40,
        left: 100
      },
      bindto: "#chart-1",
      onrendered: function() {
        // Access D3 selection
        var svg = d3.select("#chart-1 svg g");
        console.log('chart:: ', chart.internal?.state?.current?.maxTickSize?.y)

        console.log("box:.", svg.node().getBBox())

        svg.append("g").html(`<line x1="10" y1="3" x2="100" y2="3" stroke-dasharray="4" />
        <text class="svg-text-1"
            x="50"
            y="-2" 
            dominant-baseline="middle"
            text-anchor="middle"
        >
            Hola mundo
        </text>`)
        /*
        // Custom D3 code to render a diagonal line
        svg.append("line")
            .attr("x1", 0)
            .attr("y1", -20)
            .attr("x2", 200)
            .attr("y2", -20)
            .attr("stroke", "red")
            .attr("stroke-width", 12)

        
        svg.append('g').attr("x", 0).attr("y", -20)
          .attr("width",100).attr("height",20)
        */
        
      }
    })
  },[])

  return <PageContainer title="Table demo">
    <DatePicker />
    <ImageUploader />
    <div style={{ padding: '1rem' }}>
      <div id="chart-1" style={{ width: '54rem', height: '600px', "background-color": 'white' }} ></div>
    </div>
  </PageContainer>
}

