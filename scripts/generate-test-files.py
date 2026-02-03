#!/usr/bin/env python3
"""
Generate comprehensive TDMS test files for parser testing.

This script creates TDMS files with varying metadata options to test
all aspects of a TDMS parser implementation.

Requires: pip install npTDMS
"""

# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "numpy",
#     "nptdms",
# ]
# ///

import os
from datetime import datetime, timedelta

import numpy as np
from nptdms import ChannelObject, GroupObject, RootObject, TdmsWriter


def create_minimal_file(output_dir):
    """Create minimal TDMS file with single channel, no metadata."""
    filepath = os.path.join(output_dir, "01_minimal.tdms")

    with TdmsWriter(filepath) as writer:
        data = np.arange(100, dtype=np.int32)
        channel = ChannelObject("Group", "Channel", data)
        writer.write_segment([channel])

    print(f"Created: {filepath}")
    print("  - Single channel with integer data")
    print("  - Minimal metadata")
    print()


def create_basic_properties(output_dir):
    """Create file with basic file, group, and channel properties."""
    filepath = os.path.join(output_dir, "02_basic_properties.tdms")

    with TdmsWriter(filepath) as writer:
        data = np.sin(np.linspace(0, 2 * np.pi, 1000))

        root = RootObject(
            properties={
                "title": "Basic Properties Test",
                "author": "Test Suite",
                "description": "File with basic file-level properties",
            }
        )

        group = GroupObject(
            "SensorGroup",
            properties={
                "description": "Group containing sensor measurements",
                "location": "Lab A",
            },
        )

        channel = ChannelObject(
            "SensorGroup",
            "Temperature",
            data,
            properties={
                "unit_string": "°C",
                "description": "Temperature measurement",
                "sensor_id": "TEMP_001",
                "datatype": "DT_DOUBLE",
            },
        )

        writer.write_segment([root, group, channel])

    print(f"Created: {filepath}")
    print("  - File properties: title, author, description")
    print("  - Group properties: description, custom properties")
    print("  - Channel properties: units, description, sensor_id")
    print()


def create_waveform_file(output_dir):
    """Create file with complete waveform timing properties."""
    filepath = os.path.join(output_dir, "03_waveform_complete.tdms")

    # Generate waveform sampled at 10 kHz over 1 second
    sampling_rate = 10000  # Hz
    duration = 1.0  # seconds
    t = np.linspace(0, duration, int(sampling_rate * duration))
    frequency = 50  # Hz
    amplitude = 5.0  # Volts
    data = amplitude * np.sin(2 * np.pi * frequency * t)

    start_time = datetime(2024, 1, 15, 14, 30, 0)

    with TdmsWriter(filepath) as writer:
        root = RootObject(
            properties={
                "title": "Waveform Data",
                "author": "Test Suite",
                "description": "Complete waveform with timing information",
            }
        )

        channel = ChannelObject(
            "Measurements",
            "AC Voltage",
            data,
            properties={
                "unit_string": "V",
                "description": "Alternating voltage at 50 Hz",
                "wf_increment": 1.0 / sampling_rate,  # Time between samples
                "wf_start_time": start_time,
                "wf_samples": len(data),
                "wf_start_offset": 0.0,
                "wf_xname": "Time",
                "wf_xunit_string": "s",
                "wf_time_pref": "absolute",
                "monotony": "increasing",
            },
        )

        writer.write_segment([root, channel])

    print(f"Created: {filepath}")
    print("  - Full waveform timing properties")
    print("  - Start time, increment, sample count")
    print("  - X-axis naming and units")
    print(f"  - Data: {len(data)} samples at {sampling_rate} Hz")
    print()


def create_multi_channel_file(output_dir):
    """Create file with multiple channels and groups."""
    filepath = os.path.join(output_dir, "04_multi_channel.tdms")

    n_samples = 1000
    t = np.linspace(0, 10, n_samples)

    with TdmsWriter(filepath) as writer:
        root = RootObject(
            properties={
                "title": "Multi-Channel Measurement",
                "author": "Test Suite",
                "acquisition_date": datetime(2024, 1, 15),
            }
        )

        # First group with voltage measurements
        channels = [
            ChannelObject(
                "Voltage",
                "Phase_A",
                230 * np.sin(2 * np.pi * 50 * t + 0),
                properties={"unit_string": "V", "phase": 0},
            ),
            ChannelObject(
                "Voltage",
                "Phase_B",
                230 * np.sin(2 * np.pi * 50 * t - 2 * np.pi / 3),
                properties={"unit_string": "V", "phase": 120},
            ),
            ChannelObject(
                "Voltage",
                "Phase_C",
                230 * np.sin(2 * np.pi * 50 * t + 2 * np.pi / 3),
                properties={"unit_string": "V", "phase": 240},
            ),
        ]

        # Second group with current measurements
        channels.extend(
            [
                ChannelObject(
                    "Current",
                    "Phase_A",
                    50 * np.sin(2 * np.pi * 50 * t + 0.1),
                    properties={"unit_string": "A", "phase": 0},
                ),
                ChannelObject(
                    "Current",
                    "Phase_B",
                    50 * np.sin(2 * np.pi * 50 * t - 2 * np.pi / 3 + 0.1),
                    properties={"unit_string": "A", "phase": 120},
                ),
            ]
        )

        writer.write_segment([root] + channels)

    print(f"Created: {filepath}")
    print("  - 2 groups: Voltage, Current")
    print("  - 3 voltage channels, 2 current channels")
    print("  - Different properties per channel")
    print()


def create_multi_data_type_file(output_dir):
    """Create file with various data types."""
    filepath = os.path.join(output_dir, "05_multi_data_types.tdms")

    with TdmsWriter(filepath) as writer:
        root = RootObject(
            properties={
                "description": "File testing various data types",
                "version": 1,
                "enabled": True,
            }
        )

        channels = [
            ChannelObject(
                "Types",
                "Int32_Data",
                np.arange(100, dtype=np.int32),
                properties={"datatype": "DT_INT32"},
            ),
            ChannelObject(
                "Types",
                "Float_Data",
                np.linspace(0, 100, 100, dtype=np.float32),
                properties={"datatype": "DT_SINGLE"},
            ),
            ChannelObject(
                "Types",
                "Double_Data",
                np.linspace(0, 100, 100, dtype=np.float64),
                properties={"datatype": "DT_DOUBLE"},
            ),
            ChannelObject(
                "Types",
                "Bool_Data",
                np.array([True, False] * 50, dtype=np.bool_),
                properties={"datatype": "DT_BOOLEAN"},
            ),
        ]

        writer.write_segment([root] + channels)

    print(f"Created: {filepath}")
    print("  - Int32 data")
    print("  - Float (single precision) data")
    print("  - Double (double precision) data")
    print("  - Boolean data")
    print()


def create_multi_segment_file(output_dir):
    """Create file with multiple segments written sequentially."""
    filepath = os.path.join(output_dir, "06_multi_segment.tdms")

    with TdmsWriter(filepath) as writer:
        # Simulate writing data over time in multiple segments
        for segment_num in range(3):
            t = np.linspace(segment_num, segment_num + 1, 100)
            data = np.sin(2 * np.pi * t) + 0.1 * np.random.randn(100)

            root = RootObject(
                properties={
                    "segment_number": segment_num,
                    "timestamp": datetime.now() + timedelta(seconds=segment_num),
                }
            )

            channel = ChannelObject(
                "TimeSeries",
                "Sensor",
                data,
                properties={"segment": segment_num, "unit_string": "mV"},
            )

            writer.write_segment([root, channel])

    print(f"Created: {filepath}")
    print("  - 3 segments written sequentially")
    print("  - Each segment has metadata")
    print("  - Simulates continuous data acquisition")
    print()


def create_complex_properties_file(output_dir):
    """Create file with complex and custom property types."""
    filepath = os.path.join(output_dir, "07_complex_properties.tdms")

    with TdmsWriter(filepath) as writer:
        root = RootObject(
            properties={
                "title": "Complex Properties Test",
                "version": 2,
                "calibration_date": datetime(2024, 1, 1),
                "software_version": "1.2.3",
                "measurement_type": "Vibration Analysis",
            }
        )

        data = np.random.randn(1000)

        channel = ChannelObject(
            "Analysis",
            "Acceleration",
            data,
            properties={
                "unit_string": "m/s²",
                "description": "Acceleration measurements",
                "sensor_type": "Accelerometer",
                "serial_number": "ACC-12345",
                "calibration_factor": 100.5,
                "sensitivity": 0.001,
                "frequency_response_start": 1.0,
                "frequency_response_end": 10000.0,
                "sample_rate": 51200,
                "recording_duration": 10.0,
                "data_quality": "Good",
                "temperature_at_measurement": 23.5,
            },
        )

        writer.write_segment([root, channel])

    print(f"Created: {filepath}")
    print("  - Multiple property types: string, int, float, datetime")
    print("  - Real-world measurement metadata")
    print("  - Calibration and sensor information")
    print()


def create_hierarchical_structure(output_dir):
    """Create file demonstrating deep hierarchy."""
    filepath = os.path.join(output_dir, "08_hierarchical.tdms")

    with TdmsWriter(filepath) as writer:
        root = RootObject(
            properties={
                "facility": "Test Lab",
                "building": "Building A",
                "description": "Hierarchical data structure",
            }
        )

        channels = []

        # Create multiple groups with multiple channels each
        for group_num in range(3):
            group_name = f"Station_{group_num + 1}"
            for channel_num in range(3):
                channel_name = f"Sensor_{channel_num + 1}"
                data = np.random.randn(100) + group_num

                channel = ChannelObject(
                    group_name,
                    channel_name,
                    data,
                    properties={
                        "station": group_num + 1,
                        "sensor": channel_num + 1,
                        "unit_string": "mV",
                        "location": f"Position_{channel_num + 1}",
                    },
                )
                channels.append(channel)

        writer.write_segment([root] + channels)

    print(f"Created: {filepath}")
    print("  - 3 groups with 3 channels each")
    print("  - Hierarchical organization")
    print("  - Each object has descriptive properties")
    print()


def create_daqmx_raw_data_file(output_dir):
    """Create file with DAQmx raw data format."""
    filepath = os.path.join(output_dir, "09_daqmx_raw_data.tdms")

    with TdmsWriter(filepath) as writer:
        root = RootObject(
            properties={
                "title": "DAQmx Raw Data Format",
                "description": "Raw analog input data from DAQmx",
                "daqmx_format": True,
            }
        )

        # DAQmx raw data with conversion metadata
        # Raw counts from hardware
        raw_data = np.arange(1000, dtype=np.int16)

        # DAQmx scaling information
        channel = ChannelObject(
            "Analog Input",
            "AI0",
            raw_data,
            properties={
                "DAQmx_RawDataFormat": True,
                "DAQmx_RawDataReadSize": len(raw_data),
                "unit_string": "V",
                "description": "Raw analog input channel 0",
                "DAQmx_ChanType": "Analog Input",
                "DAQmx_TerminalConfig": "Default",
                "DAQmx_MinimumValue": -10.0,
                "DAQmx_MaximumValue": 10.0,
                "resolution_bits": 16,
                "DAQmx_Sensitivity": 1.0,
                "DAQmx_SensitivityUnits": "mVolts/LSB",
            },
        )

        writer.write_segment([root, channel])

    print(f"Created: {filepath}")
    print("  - DAQmx raw data format")
    print("  - Raw counts to physical units conversion")
    print("  - Hardware scaling parameters")
    print()


def create_daqmx_with_scalers(output_dir):
    """Create file with DAQmx scalers for conversion."""
    filepath = os.path.join(output_dir, "10_daqmx_with_scalers.tdms")

    with TdmsWriter(filepath) as writer:
        root = RootObject(
            properties={
                "title": "DAQmx Data with Scalers",
                "description": "Data with applied scaling",
                "hardware": "NI-DAQmx Device",
            }
        )

        # Raw data
        raw_data = np.arange(0, 4096, dtype=np.int32)

        # Scaled data (converted from raw)
        # Example: 12-bit DAQ converting to voltage range
        scaled_data = (raw_data / 4096.0) * 10.0 - 5.0  # -5V to +5V

        channels = [
            ChannelObject(
                "DAQmx_Channels",
                "AI0_Raw",
                raw_data,
                properties={
                    "DAQmx_RawDataFormat": True,
                    "unit_string": "counts",
                    "description": "Raw ADC counts",
                    "bit_width": 12,
                    "DAQmx_ConversionFunction": "LinearScaling",
                },
            ),
            ChannelObject(
                "DAQmx_Channels",
                "AI0_Scaled",
                scaled_data,
                properties={
                    "DAQmx_ScaledDataFormat": True,
                    "unit_string": "V",
                    "description": "Scaled voltage output",
                    "DAQmx_ScalingLinearSlope": 10.0 / 4096.0,
                    "DAQmx_ScalingLinearIntercept": -5.0,
                    "DAQmx_RawDataChannel": "AI0_Raw",
                },
            ),
        ]

        writer.write_segment([root] + channels)

    print(f"Created: {filepath}")
    print("  - Raw and scaled data channels")
    print("  - Linear scaling parameters")
    print("  - Cross-channel references")
    print()


def create_daqmx_polynomial_scaling(output_dir):
    """Create file with polynomial scaling."""
    filepath = os.path.join(output_dir, "11_daqmx_polynomial_scaling.tdms")

    with TdmsWriter(filepath) as writer:
        root = RootObject(
            properties={
                "title": "DAQmx Polynomial Scaling",
                "description": "Data with polynomial conversion",
                "hardware": "Temperature Sensor with Calibration",
            }
        )

        # Raw sensor data
        raw_data = np.linspace(0, 4095, 1000, dtype=np.float64)

        # Apply polynomial: T = a0 + a1*V + a2*V^2 + a3*V^3
        # Example calibration coefficients for a thermistor
        coefficients = [20.0, 15.5, -0.05, 0.0001]
        voltage = raw_data / 4096.0 * 5.0  # Convert to 0-5V
        scaled_data = (
            coefficients[0]
            + coefficients[1] * voltage
            + coefficients[2] * voltage**2
            + coefficients[3] * voltage**3
        )

        channels = [
            ChannelObject(
                "Sensors",
                "Temperature_Raw",
                raw_data,
                properties={
                    "DAQmx_RawDataFormat": True,
                    "unit_string": "counts",
                    "description": "Raw ADC counts from thermistor",
                },
            ),
            ChannelObject(
                "Sensors",
                "Temperature_Calibrated",
                scaled_data,
                properties={
                    "unit_string": "°C",
                    "description": "Temperature after polynomial scaling",
                    "DAQmx_ScalingType": "Polynomial",
                    "DAQmx_ScalingPolynomialCoefficient0": coefficients[0],
                    "DAQmx_ScalingPolynomialCoefficient1": coefficients[1],
                    "DAQmx_ScalingPolynomialCoefficient2": coefficients[2],
                    "DAQmx_ScalingPolynomialCoefficient3": coefficients[3],
                    "DAQmx_ScalingInputUnit": "V",
                    "DAQmx_CalibrationDate": datetime(2024, 1, 1),
                    "DAQmx_CalibrationTemperature": 20.0,
                },
            ),
        ]

        writer.write_segment([root] + channels)

    print(f"Created: {filepath}")
    print("  - Polynomial scaling (degree 3)")
    print("  - Calibration metadata")
    print("  - Sensor-specific parameters")
    print()


def create_daqmx_synchronized_channels(output_dir):
    """Create file with synchronized DAQmx channels."""
    filepath = os.path.join(output_dir, "12_daqmx_synchronized.tdms")

    with TdmsWriter(filepath) as writer:
        root = RootObject(
            properties={
                "title": "DAQmx Synchronized Acquisition",
                "description": "Multiple channels acquired synchronously",
                "hardware": "NI-DAQmx Synchronized Task",
                "clock_source": "OnboardClock",
                "sample_rate": 50000,
                "acquisition_date": datetime(2024, 1, 15, 10, 30, 0),
            }
        )

        # Create synchronized channels with identical timing
        t = np.linspace(0, 1, 50000)
        start_time = datetime(2024, 1, 15, 10, 30, 0)

        channels = []
        for i in range(4):
            # Slightly different frequency for each channel
            frequency = 1000 + i * 100  # 1kHz, 1.1kHz, 1.2kHz, 1.3kHz
            data = np.sin(2 * np.pi * frequency * t)

            channel = ChannelObject(
                "Synchronized_Acquisition",
                f"AI{i}",
                data,
                properties={
                    "unit_string": "V",
                    "description": f"Analog input channel {i}",
                    "DAQmx_SampleRate": 50000,
                    "DAQmx_TerminalConfig": "SingleEnded",
                    "DAQmx_VoltageRange_Min": -10.0,
                    "DAQmx_VoltageRange_Max": 10.0,
                    "wf_increment": 1.0 / 50000,
                    "wf_start_time": start_time,
                    "wf_samples": len(data),
                    "DAQmx_ChanType": "Analog Input",
                    "DAQmx_TerminalID": f"Dev1/ai{i}",
                },
            )
            channels.append(channel)

        writer.write_segment([root] + channels)

    print(f"Created: {filepath}")
    print("  - 4 synchronized analog input channels")
    print("  - Identical sample rate and timing")
    print("  - Hardware configuration parameters")
    print()


def create_daqmx_mixed_io(output_dir):
    """Create file with mixed DAQmx input/output data."""
    filepath = os.path.join(output_dir, "13_daqmx_mixed_io.tdms")

    with TdmsWriter(filepath) as writer:
        root = RootObject(
            properties={
                "title": "DAQmx Mixed I/O",
                "description": "Analog input, output, and digital channels",
                "hardware": "NI-DAQmx Device",
                "task_name": "MixedIOTask",
            }
        )

        t = np.linspace(0, 1, 1000)

        # Analog inputs
        ai_channels = [
            ChannelObject(
                "Analog_Inputs",
                "AI0",
                np.sin(2 * np.pi * 10 * t),
                properties={
                    "unit_string": "V",
                    "DAQmx_ChanType": "Analog Input",
                    "DAQmx_InputRange_Min": -10.0,
                    "DAQmx_InputRange_Max": 10.0,
                },
            ),
            ChannelObject(
                "Analog_Inputs",
                "AI1",
                np.cos(2 * np.pi * 10 * t),
                properties={
                    "unit_string": "V",
                    "DAQmx_ChanType": "Analog Input",
                    "DAQmx_InputRange_Min": -10.0,
                    "DAQmx_InputRange_Max": 10.0,
                },
            ),
        ]

        # Analog outputs (control signal)
        ao_channels = [
            ChannelObject(
                "Analog_Outputs",
                "AO0",
                np.sin(2 * np.pi * 5 * t),
                properties={
                    "unit_string": "V",
                    "DAQmx_ChanType": "Analog Output",
                    "DAQmx_OutputRange_Min": -5.0,
                    "DAQmx_OutputRange_Max": 5.0,
                    "description": "Control signal to external system",
                },
            )
        ]

        # Digital I/O
        do_channels = [
            ChannelObject(
                "Digital_IO",
                "DO0",
                np.array([True, False] * 500, dtype=np.bool_),
                properties={
                    "unit_string": "Boolean",
                    "DAQmx_ChanType": "Digital Output",
                    "description": "Control line",
                },
            ),
            ChannelObject(
                "Digital_IO",
                "DI0",
                np.array([False, True] * 500, dtype=np.bool_),
                properties={
                    "unit_string": "Boolean",
                    "DAQmx_ChanType": "Digital Input",
                    "description": "Status line",
                },
            ),
        ]

        writer.write_segment([root] + ai_channels + ao_channels + do_channels)

    print(f"Created: {filepath}")
    print("  - Analog inputs (2 channels)")
    print("  - Analog outputs (1 channel)")
    print("  - Digital I/O (2 channels)")
    print("  - Mixed channel types and ranges")
    print()


def main():
    output_dir = "./data/"
    os.makedirs(output_dir, exist_ok=True)

    print("=" * 60)
    print("TDMS Test File Generation")
    print("=" * 60)
    print()

    try:
        create_minimal_file(output_dir)
        create_basic_properties(output_dir)
        create_waveform_file(output_dir)
        create_multi_channel_file(output_dir)
        create_multi_data_type_file(output_dir)
        create_multi_segment_file(output_dir)
        create_complex_properties_file(output_dir)
        create_hierarchical_structure(output_dir)
        create_daqmx_raw_data_file(output_dir)
        create_daqmx_with_scalers(output_dir)
        create_daqmx_polynomial_scaling(output_dir)
        create_daqmx_synchronized_channels(output_dir)
        create_daqmx_mixed_io(output_dir)

        print("=" * 60)
        print(f"Successfully created 13 test files in: {output_dir}")
        print("=" * 60)
        print()
        print("Test Files Summary:")
        print("  01_minimal.tdms - Minimal structure, no metadata")
        print("  02_basic_properties.tdms - File/group/channel properties")
        print("  03_waveform_complete.tdms - Full waveform timing info")
        print("  04_multi_channel.tdms - Multiple channels and groups")
        print("  05_multi_data_types.tdms - Various data types")
        print("  06_multi_segment.tdms - Sequential segments")
        print("  07_complex_properties.tdms - Complex metadata")
        print("  08_hierarchical.tdms - Deep hierarchy")
        print("  09_daqmx_raw_data.tdms - DAQmx raw format")
        print("  10_daqmx_with_scalers.tdms - DAQmx linear scaling")
        print("  11_daqmx_polynomial_scaling.tdms - Polynomial conversion")
        print("  12_daqmx_synchronized.tdms - Synchronized multi-channel")
        print("  13_daqmx_mixed_io.tdms - Analog and digital I/O")

    except ImportError:
        print("ERROR: npTDMS not installed")
        print()
        print("Install with: pip install npTDMS")
        print()
        print("This script will then generate comprehensive test files")
        return 1

    return 0


if __name__ == "__main__":
    exit(main())
