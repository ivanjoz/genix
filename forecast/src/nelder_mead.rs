/// Maps an unbounded optimizer coordinate into `[low, high]` (logistic), so the simplex can roam
/// freely while every model parameter stays in its valid range. Coordinate 0 is the midpoint.
pub fn to_bounds(coordinate: f64, low: f64, high: f64) -> f64 {
    low + (high - low) / (1.0 + (-coordinate).exp())
}

/// Minimizes `cost` with the Nelder-Mead simplex (standard coefficients: reflect 1, expand 2,
/// contract 0.5, shrink 0.5), from `start` with an initial edge of `step`. Stops after
/// `max_evaluations` calls of `cost` or once the simplex's costs agree.
pub fn minimize<const N: usize>(cost: impl Fn(&[f64; N]) -> f64, start: [f64; N], step: f64, max_evaluations: u32) -> [f64; N] {
    let mut simplex: Vec<([f64; N], f64)> = Vec::with_capacity(N + 1);
    simplex.push((start, cost(&start)));
    for axis in 0..N {
        let mut vertex = start;
        vertex[axis] += step;
        simplex.push((vertex, cost(&vertex)));
    }
    let mut evaluations = N as u32 + 1;

    while evaluations < max_evaluations {
        simplex.sort_by(|left, right| left.1.total_cmp(&right.1));
        let best_cost = simplex[0].1;
        let (worst_vertex, worst_cost) = simplex[N];
        if (worst_cost - best_cost).abs() <= 1e-12 * (1.0 + best_cost.abs()) {
            break;
        }

        let mut centroid = [0.0; N];
        for (vertex, _) in &simplex[..N] {
            for axis in 0..N {
                centroid[axis] += vertex[axis] / N as f64;
            }
        }
        // The point at `t` along the line from the centroid through the worst vertex.
        let along = |t: f64| {
            let mut point = [0.0; N];
            for axis in 0..N {
                point[axis] = centroid[axis] + t * (worst_vertex[axis] - centroid[axis]);
            }
            point
        };

        let reflected = along(-1.0);
        let reflected_cost = cost(&reflected);
        evaluations += 1;
        if reflected_cost < best_cost {
            let expanded = along(-2.0);
            let expanded_cost = cost(&expanded);
            evaluations += 1;
            simplex[N] = if expanded_cost < reflected_cost { (expanded, expanded_cost) } else { (reflected, reflected_cost) };
            continue;
        }
        if reflected_cost < simplex[N - 1].1 {
            simplex[N] = (reflected, reflected_cost);
            continue;
        }
        // Contract on the side of whichever of the worst and the reflected point is better.
        let contracted = if reflected_cost < worst_cost { along(-0.5) } else { along(0.5) };
        let contracted_cost = cost(&contracted);
        evaluations += 1;
        if contracted_cost < worst_cost.min(reflected_cost) {
            simplex[N] = (contracted, contracted_cost);
            continue;
        }
        let best_vertex = simplex[0].0;
        for (vertex, vertex_cost) in simplex.iter_mut().skip(1) {
            for axis in 0..N {
                vertex[axis] = best_vertex[axis] + 0.5 * (vertex[axis] - best_vertex[axis]);
            }
            *vertex_cost = cost(vertex);
        }
        evaluations += N as u32;
    }
    simplex.sort_by(|left, right| left.1.total_cmp(&right.1));
    simplex[0].0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn finds_the_minimum_of_a_bowl() {
        let minimum = minimize(|point: &[f64; 2]| (point[0] - 1.0).powi(2) + 3.0 * (point[1] + 2.0).powi(2), [0.0, 0.0], 1.0, 500);
        assert!((minimum[0] - 1.0).abs() < 1e-4);
        assert!((minimum[1] + 2.0).abs() < 1e-4);
    }

    #[test]
    fn bounds_map_to_the_range() {
        assert!((to_bounds(0.0, 0.2, 0.6) - 0.4).abs() < 1e-12);
        assert!(to_bounds(50.0, 0.2, 0.6) <= 0.6);
        assert!(to_bounds(-50.0, 0.2, 0.6) >= 0.2);
    }
}
