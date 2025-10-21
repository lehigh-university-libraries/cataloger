# Partial Dataset Download with Git LFS

## Problem

The full Institutional Books dataset is **~1TB**. You don't want to download all of it for testing.

## Solution

Use Git LFS sparse checkout to download only the parquet files you need.

## Quick Start

```bash
# 1. Clone repo WITHOUT downloading LFS files
GIT_LFS_SKIP_SMUDGE=1 git clone https://huggingface.co/datasets/instdin/institutional-books-1.0

cd institutional-books-1.0

# 2. Download just the files you want (each ~100MB)
git lfs pull --include="data/train-00000-of-09831.parquet"
git lfs pull --include="data/train-00001-of-09831.parquet"
git lfs pull --include="data/train-00002-of-09831.parquet"

# 3. Run evaluation
cd ..
./cataloger eval eval-ib --sample 10
```

## Download Patterns

### First 10 files (~1GB total)
```bash
git lfs pull --include="data/train-0000[0-9]-of-09831.parquet"
```

### First 100 files (~10GB total)
```bash
git lfs pull --include="data/train-000[0-9][0-9]-of-09831.parquet"
git lfs pull --include="data/train-00[0-9][0-9][0-9]-of-09831.parquet"
```

### Specific range (files 0-5)
```bash
for i in {0..5}; do
  padded=$(printf "%05d" $i)
  git lfs pull --include="data/train-${padded}-of-09831.parquet"
done
```

### Random sample (10 random files)
```bash
# Pick random numbers between 0-9830
for num in 0042 0123 0500 1234 2000 3456 5000 6789 8000 9500; do
  git lfs pull --include="data/train-${num}-of-09831.parquet"
done
```

## Complete Workflow

### Step 1: Clone without LFS

```bash
# This is fast - only downloads git metadata, not the large files
GIT_LFS_SKIP_SMUDGE=1 git clone https://huggingface.co/datasets/instdin/institutional-books-1.0

cd institutional-books-1.0
```

### Step 2: Check what's available

```bash
# List all parquet files (these are just pointers, not actual files yet)
ls -lh data/*.parquet | head -20

# You'll see tiny files (few KB) - these are LFS pointers
```

### Step 3: Download specific files

```bash
# Download first 5 files
git lfs pull --include="data/train-00000-of-09831.parquet"
git lfs pull --include="data/train-00001-of-09831.parquet"
git lfs pull --include="data/train-00002-of-09831.parquet"
git lfs pull --include="data/train-00003-of-09831.parquet"
git lfs pull --include="data/train-00004-of-09831.parquet"
```

### Step 4: Verify downloads

```bash
# Check file sizes - should be ~100MB each
ls -lh data/train-0000[0-4]-of-09831.parquet

# Should show something like:
# -rw-r--r-- 1 user staff 93M Oct 20 08:28 train-00000-of-09831.parquet
# -rw-r--r-- 1 user staff 97M Oct 20 08:29 train-00001-of-09831.parquet
```

### Step 5: Run evaluation

```bash
cd ..

# Test with first file
./cataloger eval eval-ib --dataset ./institutional-books-1.0/data/train-00000-of-09831.parquet --sample 10

# Test with multiple files
for i in {0..4}; do
  padded=$(printf "%05d" $i)
  ./cataloger eval eval-ib \
    --dataset ./institutional-books-1.0/data/train-${padded}-of-09831.parquet \
    --sample 50 \
    --output-json results_${padded}.json
done
```

## Download More Files Later

You can always download more files later:

```bash
cd institutional-books-1.0

# Download files 100-110
git lfs pull --include="data/train-001[0-1][0-9]-of-09831.parquet"

# Or specific files
git lfs pull --include="data/train-05000-of-09831.parquet"
```

## Remove Files to Save Space

If you're done with certain files:

```bash
# Remove the actual file (keeps LFS pointer)
rm data/train-00000-of-09831.parquet

# Re-download later if needed
git lfs pull --include="data/train-00000-of-09831.parquet"
```

## Check Disk Usage

```bash
# Check how much space the data directory uses
du -sh institutional-books-1.0/data/

# Check specific files
du -h institutional-books-1.0/data/train-0000[0-9]-of-09831.parquet
```

## Recommended Subsets

### Quick Test (1 file, ~100MB)
```bash
git lfs pull --include="data/train-00000-of-09831.parquet"
```

### Small Sample (10 files, ~1GB)
```bash
git lfs pull --include="data/train-0000[0-9]-of-09831.parquet"
```

### Medium Sample (100 files, ~10GB)
```bash
for i in {0..99}; do
  padded=$(printf "%05d" $i)
  git lfs pull --include="data/train-${padded}-of-09831.parquet"
done
```

### Diverse Sample (100 files spread across dataset, ~10GB)
```bash
# Every 100th file for diversity
for i in {0..9830..100}; do
  padded=$(printf "%05d" $i)
  git lfs pull --include="data/train-${padded}-of-09831.parquet"
done
```

## Understanding File Sizes

- **LFS Pointer**: ~200 bytes (what you get with `GIT_LFS_SKIP_SMUDGE`)
- **Actual Parquet**: ~100MB average per file
- **Total dataset**: ~1TB (9,831 files)

## Troubleshooting

### Files are still tiny (KB size)

**Problem**: LFS didn't actually download the file

**Solution**:
```bash
cd institutional-books-1.0
git lfs pull --include="data/train-00000-of-09831.parquet"

# Verify
ls -lh data/train-00000-of-09831.parquet  # Should be ~100MB
```

### "This repository is over its data quota"

**Problem**: HuggingFace bandwidth limit

**Solution**: Wait and try again later, or download fewer files at once

### Want to download everything after all

```bash
cd institutional-books-1.0
git lfs pull  # Downloads ALL files (warning: ~1TB!)
```

### Check what's actually downloaded

```bash
# Show only files > 1MB (actually downloaded)
find institutional-books-1.0/data -name "*.parquet" -size +1M

# Count them
find institutional-books-1.0/data -name "*.parquet" -size +1M | wc -l
```

## Advanced: Create a Custom Subset

Create a script to download a strategic sample:

```bash
#!/bin/bash
# download_sample.sh

cd institutional-books-1.0

# First 10 (early records)
git lfs pull --include="data/train-0000[0-9]-of-09831.parquet"

# Middle 10 (middle records)
git lfs pull --include="data/train-0490[0-9]-of-09831.parquet"

# Last 10 (recent records)
git lfs pull --include="data/train-0982[0-9]-of-09831.parquet"

echo "Downloaded 30 files (~3GB) spanning the full dataset"
```

## Update Default in CLI

You can update the default dataset path in the CLI to point to your downloaded file:

```bash
./cataloger eval eval-ib --dataset ./institutional-books-1.0/data/train-00000-of-09831.parquet --sample 10
```

Or set an environment variable:

```bash
export EVAL_DATASET="./institutional-books-1.0/data/train-00000-of-09831.parquet"
```

## Summary

✅ **Clone without LFS**: `GIT_LFS_SKIP_SMUDGE=1 git clone ...`
✅ **Download selectively**: `git lfs pull --include="data/train-00000-of-09831.parquet"`
✅ **Start small**: Download 1-10 files (~100MB-1GB)
✅ **Scale up**: Download more as needed
✅ **Save space**: Remove files when done, re-download later

This approach lets you work with the dataset without downloading a terabyte of data!
