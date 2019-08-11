package metric

/*
 * The concept and definion of Matrix, Series, and Point comes frome prometheus
 */

// matrixT is a slice of Seriess that implements sort.Interface and
// has a String method.
type matrixT []seriesT
