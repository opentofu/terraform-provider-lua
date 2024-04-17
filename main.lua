function main_fib (x, y)
  return (x^2 * math.sin(y))/(1 - x)
end

function main_a( input )
    local animal_sounds = {
        cat = 'meow',
        dog = 'woof',
        cow = 'moo'
    }
    return animal_sounds
end

function main()
    a = {}
    for i=-5, 5 do
      a[i] = main_a()
    end
    a[0] = "foo"
    return a
end

function echo_new(value)
    return value
end

-- return echo_new
