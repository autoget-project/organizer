from typing import List

from ..models import PlanResponse, PlanAction, Category


def simple_move_plan(category: Category, files: List[str]) -> PlanResponse:
  res = PlanResponse(plan=[])
  # files is in following format $torrent_hash/optional_dir/file
  # if only 1 file, just copy the file to category dir
  if len(files) == 1:
    f = files[0]
    last_part = f.split("/")[-1]
    res.plan.append(PlanAction(file=f, action="move", target=f"{category.name}/{last_part}"))
    return res

  file_under_hash_dir = False
  dirs_under_hash_dir = set()
  hash_dir = files[0].split("/")[0]
  for f in files:
    parts = f.split("/")
    if len(parts) == 2:
      file_under_hash_dir = True
      break

    dirs_under_hash_dir.add(parts[1])

  # if many files under $torrent_hash dir, copy the torrent_hash dir to target
  if file_under_hash_dir:
    res.plan.append(PlanAction(file=hash_dir, action="move", target=f"{category.name}/{hash_dir}"))
    return res

  # if all files in dirs, copy the dirs
  for d in dirs_under_hash_dir:
    res.plan.append(
      PlanAction(file=f"{hash_dir}/{d}", action="move", target=f"{category.name}/{d}")
    )

  return res
