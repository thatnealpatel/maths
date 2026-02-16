import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

def approx_sqrt(num):
    hi, lo = int(np.sqrt(num)) + 1, int(np.sqrt(num))
    base = np.sqrt(num)
    ibase = int(round(base,0))
    up = (base - int(base) >= 0.50)
    delta = abs(num - ibase**2)
    if delta == 0: return base # perfect square
    frac = (delta / (ibase * 2)) * ([1, -1][up])
    # print('base: {}, ibase: {}, round-up: {}, delta: {}, frac={}'.format(base, ibase, up, delta, frac))
    return ibase + frac

def error(root, approx):
    return (abs(approx - root)) / root

x = range(1, 1000, 1)
actual_range = np.sqrt(x)
approx_sqrt_range = list(map(approx_sqrt, x))
error = list(map(error, actual_range, approx_sqrt_range))

df=pd.DataFrame({'x': x, 
    'actual': actual_range,
    'approx_sqrt': approx_sqrt_range,
    'error': error})

LINE_WIDTH = 1
fig, axs = plt.subplots(2)

axs[0].plot( 'x', 'actual', data=df, marker='', color='green', linewidth=LINE_WIDTH)
axs[0].plot( 'x', 'approx_sqrt', data=df, marker='', color='blue', linewidth=LINE_WIDTH)
axs[1].plot( 'x', 'error', data=df, marker='', color='red', linewidth=LINE_WIDTH)
plt.legend()
plt.show()