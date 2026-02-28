import type { NextConfig } from "next";
import { withShiftAPI } from "@shiftapi/next";

const nextConfig: NextConfig = {};

export default withShiftAPI(nextConfig);
