function fib (x, y)
  return (x^2 * math.sin(y))/(1 - x)
end

function echo(value)
    return value
end

-- If input array contains single value, return that value, otherwise return the original array.
function normalize(arr)
  if type(arr) == "table" and next(arr) == nil then
    return {}
  end
  if type(arr) == "table" and #arr == 0 then
    return arr[0]
  end
  return arr
end
