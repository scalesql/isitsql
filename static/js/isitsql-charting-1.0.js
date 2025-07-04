
const lodash = window._; // Assign Lodash to a custom variable

function PrintServerName ( name )
{
    console.log(name)
}

function getDefaultChartOptions() {
    return {
        animation: false,
        maintainAspectRatio: false, // Allow dynamic resizing
        layout: {
            padding: {
                bottom: 50 // Add padding to the bottom for the legend
            }
        },
        scales: {
            x: {
                type: 'time', // Use time scale for x-axis
                time: {
                    unit: 'minute', // Adjust as needed
                    displayFormats: {
                        minute: 'HH:mm', // Format labels as hh:mm (24-hour clock)
                        hour: 'HH:mm'    // Format for hourly intervals
                    },
                },
                ticks: {
                    maxTicksLimit: 4, // Limit the number of x-axis ticks
                    font: {
                        size: 11
                    }
                },
                suggestedMin: Date.now() - 3600000, // Set the minimum x-axis value to 1 hour ago
            },
            y: {
                beginAtZero: true,
                min: 0,
                title: {
                    font: {
                        size: 11  
                    }
                },
                ticks: {
                    maxTicksLimit: 5,
                    font: {
                        size: 11
                    }
                }     
            },
            y1:{
                position: 'right', // Position the second Y-axis on the right
                beginAtZero: true,
                grid: {
                    drawOnChartArea: false // Prevent grid lines from overlapping
                },
                ticks: {
                    maxTicksLimit: 5,
                    font: {
                        size: 11
                    }
                },
            }
        },
        plugins: {
            legend: {
                display: true,
                position: 'bottom',
                labels: {
                    boxWidth: 15,
                    font: {
                        size: 10
                    }
                }
            }
        }
    };
}

function NewWaitsChart(whichAPI, server, container) {
    //console.log("NewWaitsChart: " + whichAPI + " " + server + " " + container)
    Chart.register(Chart.Colors);
    const defaultOptions = getDefaultChartOptions();
        
    // Chart.js options
    const chartOptions = {
        scales: {
            y: {
                title: {
                    display: true, // Enable the title
                    text: "Total Wait Time (sec)", // Set the label text
                },
                stacked: true , // Stack the bars
            },
            y1: {
                afterDataLimits: function (axis) {
                    // Dynamically match the limits of the left Y-axis
                    const yScale = axis.chart.scales.y;
                    axis.min = yScale.min;
                    axis.max = yScale.max;
                }
            },            
        },
    };

    // Merge default options with specific options
    const options = lodash.merge({}, defaultOptions, chartOptions);
    // console.log(options)

    const ctx = document.getElementById(container);
    const apiUrl = "/api/" + whichAPI + "/" + server + "?keepsort=1"
    //console.log(apiUrl)
    $.getJSON(apiUrl, function(json) {
        // Check if json.series is valid
        if (!json || !Array.isArray(json.series)) {
            console.log(`NewWaitsChart: Invalid or missing 'series' data in API response from ${apiUrl}.`);
            return; // Exit the function without drawing the chart
        }
        //console.log(json.series)

        // Create the datasets dynamically based on the series in the JSON response
        const datasets = json.series.map((series, index, array) => {
            // Check if the series has only one data point
            // one data point doesn't display so we add a second data point
            const data = series.data.length === 1
            ? [
                // Duplicate with X value 1 minute earlier
                { ...series.data[0], x: series.data[0].x - 60000 }, 
                series.data[0]
            ]
            : series.data; // Use the data as-is
            //console.log(data)
            return {
                label: series.name, // Use the name attribute for the legend
                data: data, // Use the data attribute for the chart
                borderWidth: 1,
                tension: 0.4, // Smooth line
                pointRadius: 0, // Remove the little circles
                fill: index === 0 ? true : '-1' // Fill the area under the line
            };
        });

        //console.log(datasets)
        //console.log('test')

        // Create the Chart.js chart
        new Chart(ctx, {
            type: 'line',
            data: {
                datasets: datasets // Add the dynamically created datasets
            },
            options: options
        });
    }).fail(function() {
        console.error("NewWaitsChart: Failed to load data from " + apiUrl);
    });
}

function NewDiskChart(server, container) {
    const ctx = document.getElementById(container);
    const apiUrl = "/api/disk/" + server;

    const defaultOptions = getDefaultChartOptions();
    // Chart.js options
    const chartOptions= {
        scales: {
            y: {
                title: {
                    display: true, // Enable the title
                    text: "MB per second",
                },
            },
            y1: {
                ticks: {
                    callback: function (value) {
                        if (value >= 1000) {
                            return (value / 1000) + "k"; // Convert to multiples of 1000 and append "k"
                        }
                        return value; // Return the value as-is if less than 1000
                    },
                }
            }
        }
    };

    const options = lodash.merge({}, defaultOptions, chartOptions);

    $.getJSON(apiUrl, function(json) {
        // Extract the first series data (array of x, y pairs)
        const dataReads =   json.series[0].data;
        const dataWrites =  json.series[1].data;
        const dataPLE =     json.series[2].data; 

        // Create the Chart.js chart
        new Chart(ctx, {
            type: 'line',
            data: {
                datasets: [
                    {
                        label: "MB Read/sec",
                        data: dataReads, // Wire the second series data here
                        borderWidth: 1,
                        borderColor: 'rgba(255, 99, 132, 1)',
                        backgroundColor: 'rgba(255, 99, 132, 0.2)',
                        tension: 0.4, // Smooth line
                        pointRadius: 0, // Remove the little circles
                        fill: 'origin', 
                    },
                    {
                        label: "MB Written/sec",
                        data: dataWrites, // Wire the first series data here
                        borderWidth: 2,
                        borderColor: 'rgba(128, 0, 128, 1)', // Dark gray (close to black)
                        backgroundColor: 'rgba(128, 0, 128, 0.2)', // Light gray fill
                        tension: 0.4, // Smooth line
                        pointRadius: 0, 
                        fill: false,
                    },
                    {
                        label: "Page Life Expectency (sec)",
                        data: dataPLE, // Wire the Batches/sec series data here
                        borderWidth: 2,
                        borderColor: 'rgba(0, 255, 0, 1)', // Bright green color
                        backgroundColor: 'rgba(0, 0, 0, 0)', // No fill
                        tension: 0.4, // Smooth line
                        pointRadius: 0, // Remove the little circles
                        fill: false, // No fill for this line
                        yAxisID: 'y1' // Use the new Y-axis
                    }
            ]
            },
            options: options
        });
    }).fail(function() {
        console.error("NewDiskChart: Failed to load data from " + apiUrl);
    });
}

function roundToNearestMinute(timestamp) {
    const msPerMinute = 60000; // Number of milliseconds in a minute
    return Math.round(timestamp / msPerMinute) * msPerMinute;
}

function NewCPUChart(server, container) {
    Chart.register(Chart.Colors);
    const ctx = document.getElementById(container);
    const apiUrl = "/api/cpu/" + server;

    const defaultOptions = getDefaultChartOptions();
    const chartOptions = {
        animation: false,
        scales: {
            y: {
                max: 100,
                title: {
                    display: true, // Enable the title
                    text: "CPU Usage", // Set the label text
                },
                stacked: true, 
            },
            y1: {
                ticks: {
                    maxTicksLimit: 4,
                    stepSize: 1, // Ensure only integers are displayed
                    callback: function (value) {
                        return Number.isInteger(value) ? value : null; // Display only integers
                    },
                }
            },   
        }
    };

    $.getJSON(apiUrl, function(json) {
        // Extract the first series data (array of x, y pairs)
        const otherCPU = json.series[0].data;
        const sqlcpu = json.series[1].data;
        const sqlbatches = json.series[2].data; // Batches/sec
        
        const options = lodash.merge({}, defaultOptions, chartOptions);

        new Chart(ctx, {
            type: 'line',
            data: {
                datasets: [
                    {
                        label: "SQL CPU",
                        data: sqlcpu, // Wire the second series data here
                        borderWidth: 1,
                        borderColor: 'rgba(255, 99, 132, 1)',
                        backgroundColor: 'rgba(255, 99, 132, 0.2)',
                        tension: 0.4, // Smooth line
                        pointRadius: 0, // Remove the little circles
                        fill: true, 
                        stack: 'cpu'
                    },
                    {
                        label: "Other CPU",
                        data: otherCPU, // Wire the first series data here
                        borderWidth: 1,
                        borderColor: 'rgba(75, 192, 192, 1)',
                        backgroundColor: 'rgba(75, 192, 192, 0.2)',
                        tension: 0.4, // Smooth line
                        pointRadius: 0, 
                        fill:  '-1',
                        stack: 'cpu'
                    },
                    {
                        label: "Batches/sec",
                        data: sqlbatches, // Wire the Batches/sec series data here
                        borderWidth: 2,
                        borderColor: 'rgba(0, 255, 0, 1)', // Bright green color
                        backgroundColor: 'rgba(0, 0, 0, 0)', // No fill
                        tension: 0.4, // Smooth line
                        pointRadius: 0, // Remove the little circles
                        fill: false, // No fill for this line
                        yAxisID: 'y1' // Use the new Y-axis
                    }
            ]
            },
            options: options
        });
    }).fail(function() {
        console.error("NewCPUChart: Failed to load data from " + apiUrl);
    });
}
