# Bubble Bath

A custom authentication API that secures information using a two-digit number and a hexadecimal color code as credentials.

---

## Overview

Bubble Bath is a lightweight identity and authentication service built around a unique credential scheme: instead of traditional passwords, users authenticate with a **2-digit number** (00–99) and a **hexadecimal color** (#000000–#FFFFFF). Together, these form a compact credential pair that gates access to secured information.

The combination space yields **1,677,721,600 possible pairs** (100 numbers x 16,777,216 colors), providing a meaningful key space while keeping credentials easy to remember visually and numerically.

---

## Authentication Scheme

### Credential Pair

| Component | Format | Range | Example |
|-----------|--------|-------|---------|
| Number | 2-digit integer | 00–99 | `42` |
| Color | 6-character hex code | #000000–#FFFFFF | `#7B3F00` |

A valid credential pair might look like: `42 #7B3F00`

### How It Works

1. A user registers a credential pair (number + hex color) tied to their identity
2. Secured information is locked behind this credential pair
3. To access the information, the user provides their number and color
4. The API validates the pair and returns the secured content if authenticated

---

## Project Status

This project is in the design and early development phase. Implementation details, API endpoints, and data models are being defined.

---

## License

TBD
