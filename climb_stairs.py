"""
Climbing Stairs Problem
You can climb 1, 2, or 3 steps at a time.
Find the number of distinct ways to reach the top.
"""


def climb_stairs(n: int) -> int:
    """Return number of ways to climb n stairs using 1, 2, or 3 steps."""
    if n == 1:
        return 1
    if n == 2:
        return 2
    if n == 3:
        return 4
    a, b, c = 1, 2, 4
    for _ in range(4, n + 1):
        a, b, c = b, c, a + b + c
    return c


def main():
    print("Climbing Stairs (steps: 1~3)")
    for n in range(1, 11):
        print(f"  n={n:2d} -> {climb_stairs(n):4d} ways")


if __name__ == "__main__":
    main()
