from pydantic import BaseModel
from pydantic_ai import Agent, ToolOutput

from .ai import model, setupLogfire
from .models import PlanRequest


class VRResponse(BaseModel):
  is_vr: bool


def _get_instruction() -> str:
  return """\
Task: You are an AI agent specialized in determining if adult content files are VR (Virtual Reality) content.

Your goal is to analyze file names and metadata to determine if the content is VR or not.

Please repeat the prompt back as you understand it.

Specifics:
1. Input:
   - JSON object containing:
     - "files": array of file paths.
     - "metadata": optional fields like title, description, etc.
2. **VR Detection Criteria**:
   - Look for VR indicators in filenames, metadata such as:
     - "VR", "Virtual Reality", "VirtualReal"
     - "360", "180", "SBS" (Side-by-Side)
     - "3D", "stereoscopic", "stereoscopic 3D"
     - "VRPorn", "VRBangers", "NaughtyAmericaVR", "VRSpy", "SLR", "SexLikeReal"
     - Specific VR studio names or VR formats
     - Series of JAV bango (番号) contains VR, eg. "IPVR", "DSVR", "HNVR", "JUVR", "MDVR", "SIVR"
   - Check metadata for VR-related information
   - Consider file patterns common in VR content
3. **Edge Cases**:
   - If uncertain about VR status, default to false
   - Consider both explicit VR markers and strong VR indicators
   - Multiple files should be evaluated collectively
"""


def agent() -> Agent:
  return Agent(
    name="is_vr",
    model=model(),
    instructions=_get_instruction(),
    output_type=ToolOutput(VRResponse),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "VRPorn/ExampleScene.180.sbs.mp4",
        "VRPorn/ExampleScene.360.mp4",
      ],
    )

    a = agent()
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
