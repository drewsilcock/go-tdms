"""
TDMS Test File Generator

Generates a comprehensive set of TDMS test files covering various use cases,
along with a JSON manifest describing expected values for each file.

Usage:
    uv run python generate_tdms_test_files.py [output_directory]

Outputs:
    - Multiple .tdms test files
    - manifest.json - JSON file describing all tests and expected values
"""

import json
import sys
import typing as t
from datetime import datetime
from pathlib import Path

import numpy as np
from nptdms import ChannelObject, GroupObject, RootObject, TdmsWriter

# =============================================================================
# JSON SERIALIZATION HELPERS
# =============================================================================


def numpy_to_python(obj: t.Any) -> t.Any:
    """Convert numpy types to Python native types for JSON serialization."""
    if isinstance(obj, np.ndarray):
        return numpy_array_to_list(obj)
    elif isinstance(obj, (np.integer, np.int8, np.int16, np.int32, np.int64)):
        return int(obj)
    elif isinstance(
        obj, (np.unsignedinteger, np.uint8, np.uint16, np.uint32, np.uint64)
    ):
        return int(obj)
    elif isinstance(obj, (np.floating, np.float32, np.float64)):
        if np.isnan(obj):
            return "NaN"
        elif np.isinf(obj):
            return "Inf" if obj > 0 else "-Inf"
        return float(obj)
    elif isinstance(obj, (np.complexfloating, np.complex64, np.complex128)):
        return {"real": float(obj.real), "imag": float(obj.imag)}
    elif isinstance(obj, datetime):
        return obj.isoformat()
    elif isinstance(obj, np.datetime64):
        return str(obj)
    elif isinstance(obj, dict):
        return {k: numpy_to_python(v) for k, v in obj.items()}
    elif isinstance(obj, (list, tuple)):
        return [numpy_to_python(item) for item in obj]
    return obj


def numpy_array_to_list(arr: np.ndarray) -> list:
    """Convert numpy array to list, handling special float values."""
    result = []
    for item in arr:
        if isinstance(item, (np.floating, float)):
            if np.isnan(item):
                result.append("NaN")
            elif np.isinf(item):
                result.append("Inf" if item > 0 else "-Inf")
            else:
                result.append(float(item))
        elif isinstance(item, np.complexfloating):
            result.append({"real": float(item.real), "imag": float(item.imag)})
        elif isinstance(item, (np.integer, np.unsignedinteger)):
            result.append(int(item))
        elif isinstance(item, (np.datetime64, datetime)):
            result.append(str(item))
        elif isinstance(item, (bytes, np.bytes_)):
            result.append(item.decode("utf-8"))
        else:
            result.append(item)
    return result


def get_dtype_string(arr: np.ndarray) -> str:
    """Get a standardized dtype string for the manifest."""
    dtype = arr.dtype
    dtype_map = {
        "int8": "int8",
        "int16": "int16",
        "int32": "int32",
        "int64": "int64",
        "uint8": "uint8",
        "uint16": "uint16",
        "uint32": "uint32",
        "uint64": "uint64",
        "float32": "float32",
        "float64": "float64",
        "complex64": "complex64",
        "complex128": "complex128",
    }

    dtype_str = str(dtype)
    if dtype_str in dtype_map:
        return dtype_map[dtype_str]
    elif dtype.kind == "U" or dtype.kind == "S" or dtype.kind == "O":
        return "string"
    elif dtype.kind == "M":
        return "timestamp"
    else:
        return dtype_str


# =============================================================================
# TEST MANIFEST STRUCTURE
# =============================================================================


class TestManifest:
    """Collects test metadata for JSON output."""

    def __init__(self):
        self.tests: list[dict] = []

    def add_test(self, test: dict):
        self.tests.append(test)

    def to_dict(self) -> dict:
        return {
            "version": "1.0",
            "generated": datetime.now().isoformat(),
            "description": "TDMS test file manifest with expected values",
            "tests": self.tests,
        }

    def save(self, filepath: Path):
        with open(filepath, "w", encoding="utf-8") as f:
            json.dump(self.to_dict(), f, indent=2, ensure_ascii=False)


def create_channel_info(
    group: str, channel: str, data: np.ndarray, properties: dict | None = None
) -> dict:
    """Create channel info dict for manifest."""
    return {
        "group": group,
        "channel": channel,
        "dataType": get_dtype_string(data),
        "length": len(data),
        "data": numpy_to_python(data),
        "properties": numpy_to_python(properties) if properties else {},
    }


def create_group_info(name: str, properties: dict | None = None) -> dict:
    """Create group info dict for manifest."""
    return {
        "name": name,
        "properties": numpy_to_python(properties) if properties else {},
    }


# =============================================================================
# TEST FILE GENERATORS
# =============================================================================


def generate_simple_single_channel(output_dir: Path, manifest: TestManifest):
    """Test Case 1: Simplest possible TDMS file"""
    filename = "01_simple_single_channel.tdms"
    filepath = output_dir / filename

    data = np.array([1, 2, 3, 4, 5], dtype=np.int32)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment([ChannelObject("Group", "Channel1", data)])

    manifest.add_test(
        {
            "id": 1,
            "name": "simple_single_channel",
            "filename": filename,
            "description": "Simplest TDMS file with single group and channel",
            "features": ["basic", "int32"],
            "root": {"properties": {}},
            "groups": [create_group_info("Group")],
            "channels": [create_channel_info("Group", "Channel1", data)],
        }
    )

    print(f"Created: {filepath}")


def generate_multiple_channels_same_group(output_dir: Path, manifest: TestManifest):
    """Test Case 2: Multiple channels in same group"""
    filename = "02_multiple_channels_same_group.tdms"
    filepath = output_dir / filename

    voltage = np.array([1.1, 2.2, 3.3, 4.4, 5.5], dtype=np.float64)
    current = np.array([0.1, 0.2, 0.3, 0.4, 0.5], dtype=np.float64)
    temperature = np.array([20, 21, 22, 23, 24], dtype=np.int32)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Measurements", "Voltage", voltage),
                ChannelObject("Measurements", "Current", current),
                ChannelObject("Measurements", "Temperature", temperature),
            ]
        )

    manifest.add_test(
        {
            "id": 2,
            "name": "multiple_channels_same_group",
            "filename": filename,
            "description": "Single group with multiple channels of different types",
            "features": ["basic", "float64", "int32", "multiple_channels"],
            "root": {"properties": {}},
            "groups": [create_group_info("Measurements")],
            "channels": [
                create_channel_info("Measurements", "Voltage", voltage),
                create_channel_info("Measurements", "Current", current),
                create_channel_info("Measurements", "Temperature", temperature),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_multiple_groups(output_dir: Path, manifest: TestManifest):
    """Test Case 3: Multiple groups with channels"""
    filename = "03_multiple_groups.tdms"
    filepath = output_dir / filename

    analog_props = {"Description": "Analog measurements"}
    digital_props = {"Description": "Digital signals"}

    voltage = np.array([1.0, 2.0, 3.0], dtype=np.float64)
    current = np.array([0.5, 0.6, 0.7], dtype=np.float64)
    input1 = np.array([0, 1, 0, 1], dtype=np.uint8)
    output1 = np.array([1, 0, 1, 0], dtype=np.uint8)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                GroupObject("Analog", properties=analog_props),
                ChannelObject("Analog", "Voltage", voltage),
                ChannelObject("Analog", "Current", current),
                GroupObject("Digital", properties=digital_props),
                ChannelObject("Digital", "Input1", input1),
                ChannelObject("Digital", "Output1", output1),
            ]
        )

    manifest.add_test(
        {
            "id": 3,
            "name": "multiple_groups",
            "filename": filename,
            "description": "Multiple groups with group-level properties",
            "features": ["multiple_groups", "group_properties", "float64", "uint8"],
            "root": {"properties": {}},
            "groups": [
                create_group_info("Analog", analog_props),
                create_group_info("Digital", digital_props),
            ],
            "channels": [
                create_channel_info("Analog", "Voltage", voltage),
                create_channel_info("Analog", "Current", current),
                create_channel_info("Digital", "Input1", input1),
                create_channel_info("Digital", "Output1", output1),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_all_integer_types(output_dir: Path, manifest: TestManifest):
    """Test Case 4: All integer data types"""
    filename = "04_all_integer_types.tdms"
    filepath = output_dir / filename

    int8_data = np.array([-128, 0, 127], dtype=np.int8)
    int16_data = np.array([-32768, 0, 32767], dtype=np.int16)
    int32_data = np.array([-2147483648, 0, 2147483647], dtype=np.int32)
    int64_data = np.array(
        [-9223372036854775808, 0, 9223372036854775807], dtype=np.int64
    )
    uint8_data = np.array([0, 128, 255], dtype=np.uint8)
    uint16_data = np.array([0, 32768, 65535], dtype=np.uint16)
    uint32_data = np.array([0, 2147483648, 4294967295], dtype=np.uint32)
    uint64_data = np.array(
        [0, 9223372036854775808, 18446744073709551615], dtype=np.uint64
    )

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Integers", "int8", int8_data),
                ChannelObject("Integers", "int16", int16_data),
                ChannelObject("Integers", "int32", int32_data),
                ChannelObject("Integers", "int64", int64_data),
                ChannelObject("Integers", "uint8", uint8_data),
                ChannelObject("Integers", "uint16", uint16_data),
                ChannelObject("Integers", "uint32", uint32_data),
                ChannelObject("Integers", "uint64", uint64_data),
            ]
        )

    manifest.add_test(
        {
            "id": 4,
            "name": "all_integer_types",
            "filename": filename,
            "description": "All integer data types with min/max values",
            "features": [
                "data_types",
                "int8",
                "int16",
                "int32",
                "int64",
                "uint8",
                "uint16",
                "uint32",
                "uint64",
            ],
            "root": {"properties": {}},
            "groups": [create_group_info("Integers")],
            "channels": [
                create_channel_info("Integers", "int8", int8_data),
                create_channel_info("Integers", "int16", int16_data),
                create_channel_info("Integers", "int32", int32_data),
                create_channel_info("Integers", "int64", int64_data),
                create_channel_info("Integers", "uint8", uint8_data),
                create_channel_info("Integers", "uint16", uint16_data),
                create_channel_info("Integers", "uint32", uint32_data),
                create_channel_info("Integers", "uint64", uint64_data),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_all_float_types(output_dir: Path, manifest: TestManifest):
    """Test Case 5: All floating point data types"""
    filename = "05_all_float_types.tdms"
    filepath = output_dir / filename

    float32_data = np.array(
        [1.5, -2.5, 3.14159, np.inf, -np.inf, np.nan], dtype=np.float32
    )
    float64_data = np.array(
        [1.5, -2.5, 3.14159265358979, np.inf, -np.inf, np.nan], dtype=np.float64
    )
    float32_sci = np.array([1e-38, 1e38, 1.17549435e-38], dtype=np.float32)
    float64_sci = np.array([1e-308, 1e308, 2.2250738585072014e-308], dtype=np.float64)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Floats", "float32", float32_data),
                ChannelObject("Floats", "float64", float64_data),
                ChannelObject("Floats", "float32_scientific", float32_sci),
                ChannelObject("Floats", "float64_scientific", float64_sci),
            ]
        )

    manifest.add_test(
        {
            "id": 5,
            "name": "all_float_types",
            "filename": filename,
            "description": "Float types including special values (inf, NaN)",
            "features": ["data_types", "float32", "float64", "special_values"],
            "root": {"properties": {}},
            "groups": [create_group_info("Floats")],
            "channels": [
                create_channel_info("Floats", "float32", float32_data),
                create_channel_info("Floats", "float64", float64_data),
                create_channel_info("Floats", "float32_scientific", float32_sci),
                create_channel_info("Floats", "float64_scientific", float64_sci),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_string_data(output_dir: Path, manifest: TestManifest):
    """Test Case 6: String data"""
    filename = "06_string_data.tdms"
    filepath = output_dir / filename

    ascii_data = np.array(["Hello", "World", "Test", "Data"])
    unicode_data = np.array(["Héllo", "Wörld", "日本語", "中文"])
    special_data = np.array(
        ["Line1\nLine2", "Tab\tSeparated", 'Quote"Test', "Slash\\Path"]
    )
    empty_data = np.array(["First", "", "Third", ""])

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Strings", "ascii", ascii_data),
                ChannelObject("Strings", "unicode", unicode_data),
                ChannelObject("Strings", "special_chars", special_data),
                ChannelObject("Strings", "with_empty", empty_data),
            ]
        )

    manifest.add_test(
        {
            "id": 6,
            "name": "string_data",
            "filename": filename,
            "description": "String data including unicode and special characters",
            "features": ["data_types", "string", "unicode", "special_chars"],
            "root": {"properties": {}},
            "groups": [create_group_info("Strings")],
            "channels": [
                create_channel_info("Strings", "ascii", ascii_data),
                create_channel_info("Strings", "unicode", unicode_data),
                create_channel_info("Strings", "special_chars", special_data),
                create_channel_info("Strings", "with_empty", empty_data),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_timestamp_data(output_dir: Path, manifest: TestManifest):
    """Test Case 7: Timestamp data"""
    filename = "07_timestamp_data.tdms"
    filepath = output_dir / filename

    timestamps = np.array(
        [
            np.datetime64("2020-01-01T00:00:00"),
            np.datetime64("2021-06-15T12:30:45"),
            np.datetime64("2022-12-31T23:59:59"),
            np.datetime64("1970-01-01T00:00:00"),
            np.datetime64("2030-07-04T18:00:00"),
        ]
    )

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Time", "timestamps", timestamps),
            ]
        )

    manifest.add_test(
        {
            "id": 7,
            "name": "timestamp_data",
            "filename": filename,
            "description": "Timestamp data with various datetime values",
            "features": ["data_types", "timestamp"],
            "root": {"properties": {}},
            "groups": [create_group_info("Time")],
            "channels": [
                create_channel_info("Time", "timestamps", timestamps),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_complex_data(output_dir: Path, manifest: TestManifest):
    """Test Case 8: Complex number data"""
    filename = "08_complex_data.tdms"
    filepath = output_dir / filename

    complex64_data = np.array([1 + 2j, 3 + 4j, 5 + 6j], dtype=np.complex64)
    complex128_data = np.array(
        [1.5 + 2.5j, 3.5 + 4.5j, 5.5 + 6.5j], dtype=np.complex128
    )

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Complex", "complex64", complex64_data),
                ChannelObject("Complex", "complex128", complex128_data),
            ]
        )

    manifest.add_test(
        {
            "id": 8,
            "name": "complex_data",
            "filename": filename,
            "description": "Complex number data types",
            "features": ["data_types", "complex64", "complex128"],
            "root": {"properties": {}},
            "groups": [create_group_info("Complex")],
            "channels": [
                create_channel_info("Complex", "complex64", complex64_data),
                create_channel_info("Complex", "complex128", complex128_data),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_with_properties(output_dir: Path, manifest: TestManifest):
    """Test Case 9: File, group, and channel properties"""
    filename = "09_with_properties.tdms"
    filepath = output_dir / filename

    root_properties = {
        "name": "Test File",
        "author": "Test Generator",
        "version": 1.0,
        "sample_count": 100,
    }

    group_properties = {
        "unit": "Volts",
        "sample_rate": 1000.0,
        "description": "Voltage measurements",
    }

    channel_properties = {
        "min": -10.0,
        "max": 10.0,
        "unit_string": "V",
        "display_name": "Channel 1 Voltage",
    }

    data = np.array([1.0, 2.0, 3.0, 4.0, 5.0], dtype=np.float64)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                RootObject(properties=root_properties),
                GroupObject("Measurements", properties=group_properties),
                ChannelObject(
                    "Measurements", "Voltage", data, properties=channel_properties
                ),
            ]
        )

    manifest.add_test(
        {
            "id": 9,
            "name": "with_properties",
            "filename": filename,
            "description": "Properties at root, group, and channel levels",
            "features": [
                "properties",
                "root_properties",
                "group_properties",
                "channel_properties",
            ],
            "root": {"properties": numpy_to_python(root_properties)},
            "groups": [create_group_info("Measurements", group_properties)],
            "channels": [
                create_channel_info("Measurements", "Voltage", data, channel_properties)
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_with_linear_scaling(output_dir: Path, manifest: TestManifest):
    """Test Case 10: Linear scaling properties"""
    filename = "10_linear_scaling.tdms"
    filepath = output_dir / filename

    # Linear scaling: scaled = slope * raw + intercept
    # With slope=2.0, intercept=10.0: [1,2,3,4,5] -> [12,14,16,18,20]
    raw_data = np.array([1, 2, 3, 4, 5], dtype=np.int32)

    linear_props = {
        "NI_Scaling_Status": "unscaled",
        "NI_Number_Of_Scales": 1,
        "NI_Scale[0]_Scale_Type": "Linear",
        "NI_Scale[0]_Linear_Slope": 2.0,
        "NI_Scale[0]_Linear_Y_Intercept": 10.0,
        "NI_Scale[0]_Linear_Input_Source": 0xFFFFFFFF,
        "unit_string": "mV",
    }

    expected_scaled = [12.0, 14.0, 16.0, 18.0, 20.0]

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject(
                    "Scaled", "linear_scaled", raw_data, properties=linear_props
                ),
            ]
        )

    manifest.add_test(
        {
            "id": 10,
            "name": "linear_scaling",
            "filename": filename,
            "description": "Linear scaling: scaled = slope * raw + intercept",
            "features": ["scaling", "linear_scaling"],
            "root": {"properties": {}},
            "groups": [create_group_info("Scaled")],
            "channels": [
                create_channel_info("Scaled", "linear_scaled", raw_data, linear_props)
            ],
            "scaling": {
                "linear_scaled": {
                    "type": "Linear",
                    "slope": 2.0,
                    "intercept": 10.0,
                    "expectedScaled": expected_scaled,
                }
            },
        }
    )

    print(f"Created: {filepath}")


def generate_with_polynomial_scaling(output_dir: Path, manifest: TestManifest):
    """Test Case 11: Polynomial scaling properties"""
    filename = "11_polynomial_scaling.tdms"
    filepath = output_dir / filename

    # Polynomial: scaled = c0 + c1*x + c2*x^2 + c3*x^3
    # c0=10, c1=1, c2=2, c3=3
    # x=1: 10 + 1 + 2 + 3 = 16
    # x=2: 10 + 2 + 8 + 24 = 44
    # x=3: 10 + 3 + 18 + 81 = 112
    raw_data = np.array([1, 2, 3], dtype=np.int32)

    poly_props = {
        "NI_Number_Of_Scales": 1,
        "NI_Scale[0]_Scale_Type": "Polynomial",
        "NI_Scale[0]_Polynomial_Coefficients[0]": 10.0,
        "NI_Scale[0]_Polynomial_Coefficients[1]": 1.0,
        "NI_Scale[0]_Polynomial_Coefficients[2]": 2.0,
        "NI_Scale[0]_Polynomial_Coefficients[3]": 3.0,
    }

    expected_scaled = [16.0, 44.0, 112.0]

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject(
                    "Scaled", "polynomial_scaled", raw_data, properties=poly_props
                ),
            ]
        )

    manifest.add_test(
        {
            "id": 11,
            "name": "polynomial_scaling",
            "filename": filename,
            "description": "Polynomial scaling: scaled = c0 + c1*x + c2*x^2 + c3*x^3",
            "features": ["scaling", "polynomial_scaling"],
            "root": {"properties": {}},
            "groups": [create_group_info("Scaled")],
            "channels": [
                create_channel_info("Scaled", "polynomial_scaled", raw_data, poly_props)
            ],
            "scaling": {
                "polynomial_scaled": {
                    "type": "Polynomial",
                    "coefficients": [10.0, 1.0, 2.0, 3.0],
                    "expectedScaled": expected_scaled,
                }
            },
        }
    )

    print(f"Created: {filepath}")


def generate_with_thermocouple_scaling(output_dir: Path, manifest: TestManifest):
    """Test Case 12: Thermocouple scaling"""
    filename = "12_thermocouple_scaling.tdms"
    filepath = output_dir / filename

    # Type K thermocouple voltage in microvolts
    voltage_uv = np.array([0.0, 10.0, 100.0, 1000.0], dtype=np.float64)

    type_k_props = {
        "NI_Number_Of_Scales": 1,
        "NI_Scale[0]_Scale_Type": "Thermocouple",
        "NI_Scale[0]_Thermocouple_Thermocouple_Type": 10073,
        "NI_Scale[0]_Thermocouple_Scaling_Direction": 0,
        "NI_Scale[0]_Thermocouple_Input_Source": 0xFFFFFFFF,
    }

    # Expected temperatures for Type K (approximate)
    expected_temp = [0.0, 0.251, 2.509, 24.984]

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject(
                    "Thermocouples", "Type_K", voltage_uv, properties=type_k_props
                ),
            ]
        )

    manifest.add_test(
        {
            "id": 12,
            "name": "thermocouple_scaling",
            "filename": filename,
            "description": "Thermocouple scaling (Type K, voltage to temperature)",
            "features": ["scaling", "thermocouple_scaling"],
            "root": {"properties": {}},
            "groups": [create_group_info("Thermocouples")],
            "channels": [
                create_channel_info("Thermocouples", "Type_K", voltage_uv, type_k_props)
            ],
            "scaling": {
                "Type_K": {
                    "type": "Thermocouple",
                    "thermocoupleType": 10073,
                    "direction": "voltage_to_temperature",
                    "expectedScaled": expected_temp,
                    "tolerance": 0.01,
                }
            },
        }
    )

    print(f"Created: {filepath}")


def generate_with_rtd_scaling(output_dir: Path, manifest: TestManifest):
    """Test Case 13: RTD scaling"""
    filename = "13_rtd_scaling.tdms"
    filepath = output_dir / filename

    # Voltage values (would come from RTD measurement)
    voltage = np.array([0.08, 0.10, 0.12, 0.14, 0.16], dtype=np.float64)

    rtd_props = {
        "NI_Number_Of_Scales": 1,
        "NI_Scale[0]_Scale_Type": "RTD",
        "NI_Scale[0]_RTD_Current_Excitation": 0.001,
        "NI_Scale[0]_RTD_R0_Nominal_Resistance": 100.0,
        "NI_Scale[0]_RTD_A": 0.0039083,
        "NI_Scale[0]_RTD_B": -5.775e-07,
        "NI_Scale[0]_RTD_C": -4.183e-12,
        "NI_Scale[0]_RTD_Lead_Wire_Resistance": 0.0,
        "NI_Scale[0]_RTD_Resistance_Configuration": 2,
        "NI_Scale[0]_RTD_Input_Source": 0xFFFFFFFF,
    }

    # Expected temperatures (approximate)
    expected_temp = [-50.77, 0.0, 51.57, 103.94, 157.17]

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("RTD", "PT100", voltage, properties=rtd_props),
            ]
        )

    manifest.add_test(
        {
            "id": 13,
            "name": "rtd_scaling",
            "filename": filename,
            "description": "RTD scaling (PT100, 2-wire)",
            "features": ["scaling", "rtd_scaling"],
            "root": {"properties": {}},
            "groups": [create_group_info("RTD")],
            "channels": [create_channel_info("RTD", "PT100", voltage, rtd_props)],
            "scaling": {
                "PT100": {
                    "type": "RTD",
                    "resistanceConfiguration": 2,
                    "r0": 100.0,
                    "expectedScaled": expected_temp,
                    "tolerance": 0.1,
                }
            },
        }
    )

    print(f"Created: {filepath}")


def generate_with_table_scaling(output_dir: Path, manifest: TestManifest):
    """Test Case 14: Table (lookup) scaling"""
    filename = "14_table_scaling.tdms"
    filepath = output_dir / filename

    # Input values
    raw_data = np.array([0.5, 1.0, 1.5, 2.5, 3.0, 3.5], dtype=np.float64)

    table_props = {
        "NI_Number_Of_Scales": 1,
        "NI_Scale[0]_Scale_Type": "Table",
        "NI_Scale[0]_Table_Scaled_Values_Size": 3,
        "NI_Scale[0]_Table_Scaled_Values[0]": 1.0,
        "NI_Scale[0]_Table_Scaled_Values[1]": 2.0,
        "NI_Scale[0]_Table_Scaled_Values[2]": 3.0,
        "NI_Scale[0]_Table_Pre_Scaled_Values_Size": 3,
        "NI_Scale[0]_Table_Pre_Scaled_Values[0]": 2.0,
        "NI_Scale[0]_Table_Pre_Scaled_Values[1]": 4.0,
        "NI_Scale[0]_Table_Pre_Scaled_Values[2]": 8.0,
    }

    # Expected: interpolation between table values
    expected_scaled = [2.0, 2.0, 3.0, 6.0, 8.0, 8.0]

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject(
                    "Lookup", "table_scaled", raw_data, properties=table_props
                ),
            ]
        )

    manifest.add_test(
        {
            "id": 14,
            "name": "table_scaling",
            "filename": filename,
            "description": "Table (lookup) scaling with interpolation",
            "features": ["scaling", "table_scaling"],
            "root": {"properties": {}},
            "groups": [create_group_info("Lookup")],
            "channels": [
                create_channel_info("Lookup", "table_scaled", raw_data, table_props)
            ],
            "scaling": {
                "table_scaled": {
                    "type": "Table",
                    "scaledValues": [1.0, 2.0, 3.0],
                    "preScaledValues": [2.0, 4.0, 8.0],
                    "expectedScaled": expected_scaled,
                }
            },
        }
    )

    print(f"Created: {filepath}")


def generate_chained_scaling(output_dir: Path, manifest: TestManifest):
    """Test Case 15: Chained (multiple) scaling"""
    filename = "15_chained_scaling.tdms"
    filepath = output_dir / filename

    # Input: [1, 2, 3]
    # Scale 0 (input=raw): y = 1*x + 1 -> [2, 3, 4]
    # Scale 1 (input=scale0): y = 2*x + 2 -> [6, 8, 10]
    # Scale 2 (input=scale1): y = 3*x + 3 -> [21, 27, 33]
    raw_data = np.array([1.0, 2.0, 3.0], dtype=np.float64)

    chained_props = {
        "NI_Scaling_Status": "unscaled",
        "NI_Number_Of_Scales": 3,
        "NI_Scale[0]_Scale_Type": "Linear",
        "NI_Scale[0]_Linear_Slope": 1.0,
        "NI_Scale[0]_Linear_Y_Intercept": 1.0,
        "NI_Scale[0]_Linear_Input_Source": 0xFFFFFFFF,
        "NI_Scale[1]_Scale_Type": "Linear",
        "NI_Scale[1]_Linear_Slope": 2.0,
        "NI_Scale[1]_Linear_Y_Intercept": 2.0,
        "NI_Scale[1]_Linear_Input_Source": 0,
        "NI_Scale[2]_Scale_Type": "Linear",
        "NI_Scale[2]_Linear_Slope": 3.0,
        "NI_Scale[2]_Linear_Y_Intercept": 3.0,
        "NI_Scale[2]_Linear_Input_Source": 1,
    }

    expected_scaled = [21.0, 27.0, 33.0]

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject(
                    "Chained", "multi_scale", raw_data, properties=chained_props
                ),
            ]
        )

    manifest.add_test(
        {
            "id": 15,
            "name": "chained_scaling",
            "filename": filename,
            "description": "Multiple chained linear scalings",
            "features": ["scaling", "chained_scaling"],
            "root": {"properties": {}},
            "groups": [create_group_info("Chained")],
            "channels": [
                create_channel_info("Chained", "multi_scale", raw_data, chained_props)
            ],
            "scaling": {
                "multi_scale": {
                    "type": "Chained",
                    "scales": [
                        {
                            "type": "Linear",
                            "slope": 1.0,
                            "intercept": 1.0,
                            "inputSource": "raw",
                        },
                        {
                            "type": "Linear",
                            "slope": 2.0,
                            "intercept": 2.0,
                            "inputSource": 0,
                        },
                        {
                            "type": "Linear",
                            "slope": 3.0,
                            "intercept": 3.0,
                            "inputSource": 1,
                        },
                    ],
                    "expectedScaled": expected_scaled,
                }
            },
        }
    )

    print(f"Created: {filepath}")


def generate_multiple_segments(output_dir: Path, manifest: TestManifest):
    """Test Case 16: Multiple segments"""
    filename = "16_multiple_segments.tdms"
    filepath = output_dir / filename

    seg1_data = np.array([1, 2, 3], dtype=np.int32)
    seg2_data = np.array([4, 5, 6], dtype=np.int32)
    seg3_data = np.array([7, 8, 9, 10], dtype=np.int32)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment([ChannelObject("Data", "Counter", seg1_data)])
        writer.write_segment([ChannelObject("Data", "Counter", seg2_data)])
        writer.write_segment([ChannelObject("Data", "Counter", seg3_data)])

    # Combined data from all segments
    combined_data = np.concatenate([seg1_data, seg2_data, seg3_data])

    manifest.add_test(
        {
            "id": 16,
            "name": "multiple_segments",
            "filename": filename,
            "description": "Data spread across multiple segments",
            "features": ["segments", "multiple_segments"],
            "root": {"properties": {}},
            "groups": [create_group_info("Data")],
            "channels": [create_channel_info("Data", "Counter", combined_data)],
            "segments": [
                {"data": numpy_to_python(seg1_data)},
                {"data": numpy_to_python(seg2_data)},
                {"data": numpy_to_python(seg3_data)},
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_segments_with_different_channels(output_dir: Path, manifest: TestManifest):
    """Test Case 17: Different channels in different segments"""
    filename = "17_segments_different_channels.tdms"
    filepath = output_dir / filename

    chan_a_seg1 = np.array([1.0, 2.0, 3.0], dtype=np.float64)
    chan_b_seg2 = np.array([10, 20, 30], dtype=np.int32)
    chan_a_seg3 = np.array([4.0, 5.0], dtype=np.float64)
    chan_b_seg3 = np.array([40, 50], dtype=np.int32)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment([ChannelObject("Group", "ChannelA", chan_a_seg1)])
        writer.write_segment([ChannelObject("Group", "ChannelB", chan_b_seg2)])
        writer.write_segment(
            [
                ChannelObject("Group", "ChannelA", chan_a_seg3),
                ChannelObject("Group", "ChannelB", chan_b_seg3),
            ]
        )

    combined_a = np.concatenate([chan_a_seg1, chan_a_seg3])
    combined_b = np.concatenate([chan_b_seg2, chan_b_seg3])

    manifest.add_test(
        {
            "id": 17,
            "name": "segments_different_channels",
            "filename": filename,
            "description": "Different channels appearing in different segments",
            "features": ["segments", "sparse_channels"],
            "root": {"properties": {}},
            "groups": [create_group_info("Group")],
            "channels": [
                create_channel_info("Group", "ChannelA", combined_a),
                create_channel_info("Group", "ChannelB", combined_b),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_waveform_properties(output_dir: Path, manifest: TestManifest):
    """Test Case 18: Waveform properties"""
    filename = "18_waveform_properties.tdms"
    filepath = output_dir / filename

    wf_props = {
        "wf_start_offset": 0.0,
        "wf_increment": 0.001,
        "wf_samples": 1000,
        "NI_ChannelName": "Analog Input 0",
        "unit_string": "V",
    }

    # Generate a 10 Hz sine wave at 1kHz sample rate
    t = np.arange(0, 1000) * 0.001
    data = np.sin(2 * np.pi * 10 * t).astype(np.float64)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Acquisition", "AI0", data, properties=wf_props),
            ]
        )

    manifest.add_test(
        {
            "id": 18,
            "name": "waveform_properties",
            "filename": filename,
            "description": "DAQmx-style waveform properties",
            "features": ["waveform", "waveform_properties"],
            "root": {"properties": {}},
            "groups": [create_group_info("Acquisition")],
            "channels": [create_channel_info("Acquisition", "AI0", data, wf_props)],
            "waveform": {
                "AI0": {
                    "startOffset": 0.0,
                    "increment": 0.001,
                    "samples": 1000,
                    "expectedTimeRange": [0.0, 0.999],
                }
            },
        }
    )

    print(f"Created: {filepath}")


def generate_empty_channel(output_dir: Path, manifest: TestManifest):
    """Test Case 19: Empty channel"""
    filename = "19_empty_channel.tdms"
    filepath = output_dir / filename

    empty_data = np.array([], dtype=np.float64)
    non_empty_data = np.array([1.0, 2.0, 3.0], dtype=np.float64)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Group", "EmptyChannel", empty_data),
                ChannelObject("Group", "NonEmptyChannel", non_empty_data),
            ]
        )

    manifest.add_test(
        {
            "id": 19,
            "name": "empty_channel",
            "filename": filename,
            "description": "Channel with no data",
            "features": ["edge_case", "empty_channel"],
            "root": {"properties": {}},
            "groups": [create_group_info("Group")],
            "channels": [
                create_channel_info("Group", "EmptyChannel", empty_data),
                create_channel_info("Group", "NonEmptyChannel", non_empty_data),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_large_data(output_dir: Path, manifest: TestManifest):
    """Test Case 20: Large dataset"""
    filename = "20_large_data.tdms"
    filepath = output_dir / filename

    # Use a seed for reproducibility
    np.random.seed(42)
    large_data = np.random.randn(100000).astype(np.float64)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("LargeData", "RandomValues", large_data),
            ]
        )

    # Only include summary statistics, not all 100k values
    manifest.add_test(
        {
            "id": 20,
            "name": "large_data",
            "filename": filename,
            "description": "Large dataset (100,000 samples)",
            "features": ["edge_case", "large_data"],
            "root": {"properties": {}},
            "groups": [create_group_info("LargeData")],
            "channels": [
                {
                    "group": "LargeData",
                    "channel": "RandomValues",
                    "dataType": "float64",
                    "length": 100000,
                    "data": None,  # Too large to include
                    "properties": {},
                    "statistics": {
                        "min": float(np.min(large_data)),
                        "max": float(np.max(large_data)),
                        "mean": float(np.mean(large_data)),
                        "std": float(np.std(large_data)),
                        "first10": numpy_to_python(large_data[:10]),
                        "last10": numpy_to_python(large_data[-10:]),
                    },
                }
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_special_names(output_dir: Path, manifest: TestManifest):
    """Test Case 21: Special characters in names"""
    filename = "21_special_names.tdms"
    filepath = output_dir / filename

    data1 = np.array([1, 2, 3], dtype=np.int32)
    data2 = np.array([4, 5, 6], dtype=np.int32)
    data3 = np.array([7, 8, 9], dtype=np.int32)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                GroupObject("Group With Spaces"),
                ChannelObject("Group With Spaces", "Channel-With-Dashes", data1),
                ChannelObject("Group With Spaces", "Channel_With_Underscores", data2),
                GroupObject("グループ日本語"),
                ChannelObject("グループ日本語", "チャンネル", data3),
            ]
        )

    manifest.add_test(
        {
            "id": 21,
            "name": "special_names",
            "filename": filename,
            "description": "Special characters in group and channel names",
            "features": ["edge_case", "special_characters", "unicode_names"],
            "root": {"properties": {}},
            "groups": [
                create_group_info("Group With Spaces"),
                create_group_info("グループ日本語"),
            ],
            "channels": [
                create_channel_info("Group With Spaces", "Channel-With-Dashes", data1),
                create_channel_info(
                    "Group With Spaces", "Channel_With_Underscores", data2
                ),
                create_channel_info("グループ日本語", "チャンネル", data3),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_boolean_data(output_dir: Path, manifest: TestManifest):
    """Test Case 22: Boolean data"""
    filename = "22_boolean_data.tdms"
    filepath = output_dir / filename

    bool_data = np.array([True, False, True, False, True], dtype=np.bool_)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Digital", "BooleanChannel", bool_data),
            ]
        )

    manifest.add_test(
        {
            "id": 22,
            "name": "boolean_data",
            "filename": filename,
            "description": "Boolean data type",
            "features": ["data_types", "boolean"],
            "root": {"properties": {}},
            "groups": [create_group_info("Digital")],
            "channels": [
                {
                    "group": "Digital",
                    "channel": "BooleanChannel",
                    "dataType": "boolean",
                    "length": 5,
                    "data": [True, False, True, False, True],
                    "properties": {},
                }
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_property_types(output_dir: Path, manifest: TestManifest):
    """Test Case 23: Various property data types"""
    filename = "23_property_types.tdms"
    filepath = output_dir / filename

    props = {
        "string_prop": "Hello World",
        "int32_prop": 42,
        "float64_prop": 3.14159,
        "bool_prop": True,
    }

    data = np.array([1.0, 2.0, 3.0], dtype=np.float64)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Group", "Channel", data, properties=props),
            ]
        )

    manifest.add_test(
        {
            "id": 23,
            "name": "property_types",
            "filename": filename,
            "description": "Various property data types",
            "features": ["properties", "property_types"],
            "root": {"properties": {}},
            "groups": [create_group_info("Group")],
            "channels": [create_channel_info("Group", "Channel", data, props)],
            "propertyTypes": {
                "string_prop": "string",
                "int32_prop": "int32",
                "float64_prop": "float64",
                "bool_prop": "boolean",
            },
        }
    )

    print(f"Created: {filepath}")


def generate_endian_test(output_dir: Path, manifest: TestManifest):
    """Test Case 24: Values for endianness testing"""
    filename = "24_endian_test.tdms"
    filepath = output_dir / filename

    # Values chosen to be obviously different if byte order is wrong
    int16_data = np.array([0x0102, 0x1020, 0x00FF], dtype=np.int16)
    int32_data = np.array([0x01020304, 0x10203040], dtype=np.int32)
    float64_data = np.array([1.0, 256.0, 65536.0], dtype=np.float64)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject("Endian", "int16_test", int16_data),
                ChannelObject("Endian", "int32_test", int32_data),
                ChannelObject("Endian", "float64_test", float64_data),
            ]
        )

    manifest.add_test(
        {
            "id": 24,
            "name": "endian_test",
            "filename": filename,
            "description": "Values for testing byte order handling",
            "features": ["edge_case", "endianness"],
            "root": {"properties": {}},
            "groups": [create_group_info("Endian")],
            "channels": [
                create_channel_info("Endian", "int16_test", int16_data),
                create_channel_info("Endian", "int32_test", int32_data),
                create_channel_info("Endian", "float64_test", float64_data),
            ],
        }
    )

    print(f"Created: {filepath}")


def generate_strain_scaling(output_dir: Path, manifest: TestManifest):
    """Test Case 25: Strain gauge scaling"""
    filename = "25_strain_scaling.tdms"
    filepath = output_dir / filename

    strain_voltage = np.array(
        [0.0068827, 0.0068036, 0.00688, 0.0068545, 0.0069104], dtype=np.float64
    )

    strain_props = {
        "NI_Number_Of_Scales": 1,
        "NI_Scale[0]_Scale_Type": "Strain",
        "NI_Scale[0]_Strain_Configuration": 10183,  # Quarter bridge I
        "NI_Scale[0]_Strain_Poisson_Ratio": 0.3,
        "NI_Scale[0]_Strain_Gage_Resistance": 350.0,
        "NI_Scale[0]_Strain_Lead_Wire_Resistance": 0.0,
        "NI_Scale[0]_Strain_Initial_Bridge_Voltage": 0.0,
        "NI_Scale[0]_Strain_Gage_Factor": 2.1,
        "NI_Scale[0]_Strain_Bridge_Shunt_Calibration_Gain_Adjustment": 1.0,
        "NI_Scale[0]_Strain_Voltage_Excitation": 2.5,
        "NI_Scale[0]_Strain_Input_Source": 0xFFFFFFFF,
    }

    # Expected strain values (approximate, from test_scaling.py)
    expected_strain = [
        -1.31099e-03,
        -1.29592e-03,
        -1.31048e-03,
        -1.30562e-03,
        -1.31627e-03,
    ]

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                ChannelObject(
                    "Strain", "quarter_bridge", strain_voltage, properties=strain_props
                ),
            ]
        )

    manifest.add_test(
        {
            "id": 25,
            "name": "strain_scaling",
            "filename": filename,
            "description": "Strain gauge scaling (Quarter Bridge I)",
            "features": ["scaling", "strain_scaling"],
            "root": {"properties": {}},
            "groups": [create_group_info("Strain")],
            "channels": [
                create_channel_info(
                    "Strain", "quarter_bridge", strain_voltage, strain_props
                )
            ],
            "scaling": {
                "quarter_bridge": {
                    "type": "Strain",
                    "configuration": 10183,
                    "gaugeFactor": 2.1,
                    "expectedScaled": expected_strain,
                    "tolerance": 1e-6,
                }
            },
        }
    )

    print(f"Created: {filepath}")


def generate_comprehensive_test(output_dir: Path, manifest: TestManifest):
    """Test Case 26: Comprehensive combined test"""
    filename = "26_comprehensive_test.tdms"
    filepath = output_dir / filename

    root_props = {
        "name": "Comprehensive Test",
        "version": 1,
    }

    analog_props = {"sample_rate": 1000.0}
    voltage_data = np.sin(np.linspace(0, 10 * np.pi, 100)).astype(np.float64)
    current_data = np.cos(np.linspace(0, 10 * np.pi, 100)).astype(np.float64)

    digital_data = np.array(
        [0, 1, 0, 1, 0, 1, 0, 1] * 12 + [0, 1, 0, 1], dtype=np.uint8
    )

    counter_data = np.arange(0, 100, dtype=np.uint32)

    with TdmsWriter(str(filepath)) as writer:
        writer.write_segment(
            [
                RootObject(properties=root_props),
                GroupObject("Analog", properties=analog_props),
                ChannelObject(
                    "Analog", "Voltage", voltage_data, properties={"unit_string": "V"}
                ),
                ChannelObject(
                    "Analog", "Current", current_data, properties={"unit_string": "A"}
                ),
            ]
        )
        writer.write_segment(
            [
                GroupObject("Digital"),
                ChannelObject("Digital", "Trigger", digital_data),
            ]
        )
        writer.write_segment(
            [
                GroupObject("Counter"),
                ChannelObject("Counter", "Count", counter_data),
            ]
        )

    manifest.add_test(
        {
            "id": 26,
            "name": "comprehensive_test",
            "filename": filename,
            "description": "Comprehensive test with multiple groups, segments, and data types",
            "features": ["comprehensive", "multiple_groups", "multiple_segments"],
            "root": {"properties": numpy_to_python(root_props)},
            "groups": [
                create_group_info("Analog", analog_props),
                create_group_info("Digital"),
                create_group_info("Counter"),
            ],
            "channels": [
                create_channel_info(
                    "Analog", "Voltage", voltage_data, {"unit_string": "V"}
                ),
                create_channel_info(
                    "Analog", "Current", current_data, {"unit_string": "A"}
                ),
                create_channel_info("Digital", "Trigger", digital_data),
                create_channel_info("Counter", "Count", counter_data),
            ],
        }
    )

    print(f"Created: {filepath}")


# =============================================================================
# MAIN EXECUTION
# =============================================================================


def main():
    output_dir = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("testdata")

    print(f"Generating TDMS test files in: {output_dir}")
    print("=" * 60)

    output_dir.mkdir(parents=True, exist_ok=True)

    manifest = TestManifest()

    generators = [
        generate_simple_single_channel,
        generate_multiple_channels_same_group,
        generate_multiple_groups,
        generate_all_integer_types,
        generate_all_float_types,
        generate_string_data,
        generate_timestamp_data,
        generate_complex_data,
        generate_with_properties,
        generate_with_linear_scaling,
        generate_with_polynomial_scaling,
        generate_with_thermocouple_scaling,
        generate_with_rtd_scaling,
        generate_with_table_scaling,
        generate_chained_scaling,
        generate_multiple_segments,
        generate_segments_with_different_channels,
        generate_waveform_properties,
        generate_empty_channel,
        generate_large_data,
        generate_special_names,
        generate_boolean_data,
        generate_property_types,
        generate_endian_test,
        generate_strain_scaling,
        generate_comprehensive_test,
    ]

    for generator in generators:
        try:
            generator(output_dir, manifest)
        except Exception as e:
            print(f"Error in {generator.__name__}: {e}")
            import traceback

            traceback.print_exc()

    # Save the manifest
    manifest_path = output_dir / "manifest.json"
    manifest.save(manifest_path)

    print("=" * 60)
    print(f"Generated {len(generators)} test files")
    print(f"Manifest saved to: {manifest_path}")


if __name__ == "__main__":
    main()
